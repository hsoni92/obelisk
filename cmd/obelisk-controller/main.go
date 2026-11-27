package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/adminapi"
	"github.com/hsoni92/obelisk/internal/backup"
	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/internal/history"
	"github.com/hsoni92/obelisk/internal/mailgateway"
	"github.com/hsoni92/obelisk/internal/mcpgateway"
	"github.com/hsoni92/obelisk/internal/openroutergateway"
	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/internal/restore"
	"github.com/hsoni92/obelisk/internal/vitals"
)

func main() {
	configPath := flag.String("config", config.ResolvePath(), "path to obelisk.yaml (OBELISK_CONFIG)")
	helmRoot := flag.String("helm-root", config.ResolveHelmRoot(), "repo root with deploy/helm (OBELISK_ROOT)")
	addr := flag.String("listen", ":8080", "admin API listen address")
	gatewayAddr := flag.String("gateway-listen", ":8081", "OpenRouter gateway listen address")
	mailGatewayAddr := flag.String("mail-gateway-listen", ":8082", "Mail gateway listen address")
	mcpGatewayAddr := flag.String("mcp-gateway-listen", ":8083", "MCP gateway listen address")
	reconcileEvery := flag.Duration("reconcile", 30*time.Second, "background reconcile interval")
	flag.Parse()

	config.LoadRepoDotEnv()

	// Appliance installs write the live file to /var/lib even when -config points at /etc.
	cfgFile := config.ActiveConfigPath(*configPath)
	cfgFile = config.EnsureReadablePath(cfgFile)
	if cfgFile != *configPath {
		log.Printf("config: using %s (%s not present)", cfgFile, *configPath)
	}

	gw := openroutergateway.NewServer(cfgFile)
	if err := gw.Reload(); err != nil {
		log.Printf("openrouter gateway reload: %v", err)
	}
	mgw := mailgateway.NewServer(cfgFile)
	if err := mgw.Reload(); err != nil {
		log.Printf("mail gateway reload: %v", err)
	}
	mcpgw := mcpgateway.NewServer(cfgFile, mgw)
	if err := mcpgw.Reload(); err != nil {
		log.Printf("mcp gateway reload: %v", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	bp, err := blueprint.NewRuntime(cfgFile, *helmRoot)
	if err != nil {
		log.Fatal(err)
	}
	bp.Start(ctx)
	ns := strings.TrimSpace(os.Getenv("OBELISK_NAMESPACE"))
	backupSched := backup.NewScheduleRunner(cfgFile, *helmRoot, ns)
	backupSched.Start(ctx)
	config.SetAfterSave(func(string) {
		bp.Sync.Enqueue()
		backupSched.Reload()
	})
	vitalsStore := vitals.NewStore()
	connHistory := history.Shared(config.HistoryPath(cfgFile))
	peerMonitor := peer.StartMonitor(ctx, cfgFile, connHistory)
	api := &adminapi.Server{
		ConfigPath:            cfgFile,
		HelmRoot:              *helmRoot,
		Gateway:               gw,
		GatewayListenAddr:     *gatewayAddr,
		MailGateway:           mgw,
		MailGatewayListenAddr: *mailGatewayAddr,
		MCPGateway:            mcpgw,
		MCPGatewayListenAddr:  *mcpGatewayAddr,
		Blueprint:             bp,
		Restore:               restore.New(),
		Vitals:                vitalsStore,
		Connectivity:          peerMonitor,
	}
	vitals.StartSamplerLogged(ctx, vitalsStore, vitals.SamplerConfig{
		ConfigPath: cfgFile,
		HelmRoot:   *helmRoot,
		Namespace:  ns,
	})
	srv := &http.Server{Addr: *addr, Handler: api.Handler()}
	gwSrv := &http.Server{Addr: *gatewayAddr, Handler: gw.Handler()}
	mailSrv := &http.Server{Addr: *mailGatewayAddr, Handler: mgw.Handler()}
	mcpSrv := &http.Server{Addr: *mcpGatewayAddr, Handler: mcpgw.Handler()}

	rec := controller.Reconciler{
		ConfigPath: cfgFile,
		HelmRoot:   *helmRoot,
		Namespace:  strings.TrimSpace(os.Getenv("OBELISK_NAMESPACE")),
	}
	engine := controller.NewEngine(rec)
	controller.SetGlobalEngine(engine)
	engine.Start(ctx, *reconcileEvery)
	go peer.StartAdvertiser(ctx, cfgFile, *reconcileEvery)

	go func() {
		log.Printf("admin API listening on %s (config %s, helm-root %s)", *addr, cfgFile, *helmRoot)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go func() {
		log.Printf("openrouter gateway listening on %s", *gatewayAddr)
		if err := gwSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go func() {
		log.Printf("mail gateway listening on %s", *mailGatewayAddr)
		if err := mailSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go func() {
		log.Printf("mcp gateway listening on %s", *mcpGatewayAddr)
		if err := mcpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdown)
	_ = gwSrv.Shutdown(shutdown)
	_ = mailSrv.Shutdown(shutdown)
	_ = mcpSrv.Shutdown(shutdown)
}
