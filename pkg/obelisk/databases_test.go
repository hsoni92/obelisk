package obelisk

import "testing"

func TestValidateDatabaseName(t *testing.T) {
	for _, name := range []string{"data", "app1", "analytics_v2"} {
		if err := ValidateDatabaseName(name); err != nil {
			t.Fatalf("%q: %v", name, err)
		}
	}
	if err := ValidateDatabaseName("default"); err == nil {
		t.Fatal("expected reject bootstrap name")
	}
	if err := ValidateDatabaseName("9bad"); err == nil {
		t.Fatal("expected reject")
	}
}

func TestDesiredDatabaseNames(t *testing.T) {
	inst := ServiceInstance{
		Type: "postgres", ID: "xyz",
		Credentials: InstanceCredentials{Database: BootstrapDatabaseName},
		Databases:   []string{"data", "data", "Analytics"},
	}
	got := DesiredDatabaseNames(inst)
	want := []string{"default", "data", "analytics"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestNewServiceInstanceBootstrapDB(t *testing.T) {
	inst, err := NewServiceInstance("postgres", "xyz", "App")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Credentials.Database != BootstrapDatabaseName {
		t.Fatalf("database = %q", inst.Credentials.Database)
	}
}
