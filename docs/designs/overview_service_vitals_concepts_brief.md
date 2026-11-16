# Overview service vitals

* **Author:** hsoni92
* **Date:** 2026-11-16

## Overview

Obelisk runs many workloads on one machine. The Overview page shows whether those workloads are up, and it shows backup status. It does not show which workload is using the memory, processor, or disk the machine is short of.

This feature adds a live block on that page. Each data-service instance is one row. The platform is three rows: the controller, the admin UI, and Kong. Each row has filled meters for memory, processor, and disk. The fill is that workload's share of the machine. The row at the top holds the most of whichever of those three the machine is shortest on. A stopped service stays in the list because disabling it keeps its disk. The numbers are the latest sample. They do not follow the timeline range, and they are not stored as a history.

## Analysis

Seven reviews read the same proposal: meter each service against its own memory cap and its own volume size, and sort by whichever meter is fullest. That proposal does not match this appliance.

### The limits in the proposal are not enforced

`hardware.tier` is the word `floor` or `comfort`. Nothing applies it. No container sets a memory limit. The only memory ceiling in the charts is Redis `--maxmemory 256mb`, which makes Redis drop keys. It is not the kernel limit that kills a process, and it does not change with the tier. A volume's requested size (Postgres 10Gi, Redis 2Gi, RustFS 20Gi, and the others) is a claim. On this node's local disk layout that claim is not a quota. A volume can sit at a small fraction of its claim while the machine disk is full, and the sum of the claims can be larger than the disk. k3s, container images, and the controller's own directories sit outside every service row.

Sorting by "fullest of three unlike meters" then promotes the wrong row. A Redis process near 256 MB, or a small volume near its claim, ranks above a Postgres process that is using most of the machine's RAM.

The reviews split on what to do about the missing caps. One side would build real per-service memory limits and meter against those. The other side treats that as a second project: writing limits restarts pods and can turn on Redis eviction. This brief takes the second side. The meters use the machine as the denominator. Where a real cap already exists, it can be drawn as a mark. This feature does not create caps.

### What each meter means

Memory is the workload's current use divided by the machine's memory. Processor is that workload's share of the machine over the last overview interval. Disk is bytes held by that workload's volumes, divided by the machine's disk for the fill that answers "who is filling the box." The volume's requested size is labeled as a request beside the meter.

Processor share needs two readings. A single counter is a total, not a rate. The controller keeps the previous counter and its time in memory and overwrites them on each sample. After a restart the first interval has no processor figure. Showing zero there would rank a busy service as idle.

### Sampling without stalling the page

The overview handler already loads health, backup readiness, and the timeline, and the page polls it about every 8 seconds. Disk bytes for a stopped service are not available from the pod, because disable scales the workload to zero and leaves the volume. A directory walk on that request would make the health verdict wait on the disk the page is warning about, and it would add reads on that disk.

A sampler outside the request updates one latest snapshot. The overview response attaches that snapshot. One collection runs at a time. Unmounted volumes are measured at a slower pace than the page poll, on the order of a minute, with a timeout. If a walk is still running, the page keeps the previous disk figure, refreshes memory and processor from the faster read, and still returns health, backup, and the timeline. The disk figure shows its own time when it is older than the other two. Nothing is written to the state-history log. That log records up and down transitions. Putting gauges in it would turn it into a metrics series and can punch gaps in the timeline. No size query is sent to Postgres or Redis. No metrics-server pod is added.

A failed or skipped reading is unknown. Unknown is never drawn as zero, and it does not replace the rest of the overview. Reconcile-off keeps returning components and history as it does now.

### Rows

Overview today has one platform row and one row per data-service instance. The platform release is three workloads with separate resource use: controller, admin UI, and Kong. OpenRouter and mail run inside the controller process, so they are not rows. SearXNG has two volumes (a small config claim and a data claim). The disk meter lists each volume. The row's disk share uses the sum of bytes held. The controller's host directories (`/etc/obelisk`, `/var/lib/obelisk`, `/opt/obelisk`) are appliance files, not a data volume, and they stay off the disk meter.

The rows will not add up to the machine. k3s, containerd, and backup jobs sit outside them. The block shows machine memory, processor, and disk, and the part the rows do not account for. Backup jobs are a known memory spike on the small tier and are not their own row in this feature. A spike that ends before the next sample leaves no series behind. The live sample therefore also carries restart count and the last stop reason (out of memory, evicted) so a crash between polls is still visible. Node pressure (memory, disk, process-id) is shown with the machine totals when the cluster reports it.

### How the list is ordered and drawn

Order is the workload holding the most of the resource the machine is shortest on (least free among memory, disk, and processor). A stopped service competes on retained disk only. Its memory and processor read as not running, not as healthy zeros. A missing reading sorts as unknown, above the numeric rows, so a service that could not be measured is not hidden under the ones that look fine. Equal shares keep a stable order by the existing component key, so the list does not shuffle when nothing changed.

Each meter prints the amount used, the machine total, and the percent. The meter that placed the row first is labeled in words. The section says the figures are the latest sample and do not follow the hour-range control. The page keeps the last good readings when a refresh fails. The existing progress bar and byte formatting are enough. There is no sparkline and no second range control.

The vitals travel as an optional addition on the current overview response. The component list, the summary counts, and the history stay in their current order and shape. An older admin page ignores the new field. A newer page still draws health, backup, and the timeline when the field is absent.

### What this still will not catch

A short processor spike is flattened across the poll gap. A service can be saturated on one core while its share of the whole machine stays small. A service about to hit Redis's own 256 MB ceiling will not rise to the top while the machine still has free memory; the mark on that meter is the only signal. The disk number can be about a minute old during the fill that matters, which is why its time is shown. Inodes and disk wait are left out. The failures this page is for are a full disk and a machine that is out of memory.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * Live memory, processor, and disk for each data-service instance and for the controller, admin UI, and Kong, on the Overview page.
  * Machine totals and the usage the rows do not account for.
  * Disk bytes for stopped services whose volumes are still on the machine.
  * Each of a service's volumes shown, with the row's disk share taken from the bytes they hold together.
  * Order by the largest share of the resource the machine is shortest on, with unknown readings visible and a stable tie-break.
  * Latest sample only, attached to the current overview response, sampled off the request path.
  * Restart count and last stop reason on that sample, plus node pressure when the cluster reports it.
  * A missing reading stays unknown and leaves the rest of the page intact.
* **Out-of-Scope Non-Goals:**
  * Writing memory limits, changing Redis `maxmemory`, resizing volumes, or starting a stopped pod in order to measure it.
  * A metrics history, sparklines, Prometheus, Grafana, or a metrics-server pod.
  * A second poll, a second endpoint, or a new API version.
  * Changing the component list, summary counts, or the state-history log.
  * Rows for OpenRouter, mail, backup jobs, k3s, or containerd.
  * Per-core saturation, inode counts, and disk wait.
  * Treating a volume's requested size as a quota.

## Alternatives Weighed

* **Sort by the fullest of each service's own memory cap, volume size, and processor share.** This was the anchored hypothesis. Set aside after review: those caps are not configured, and a volume request is not a quota, so the top row would not be the workload filling the machine.
* **Build tier memory profiles and enforce container limits first, then meter against them.** Set aside for this feature: it restarts pods and can change Redis eviction. It is a separate change. The meter can gain a cap mark later if those limits exist.
* **Host gauges only, with no per-service rows.** Set aside by the author: the operator needs to see which service to act on.
* **Walk every volume inside the overview handler on each 8-second poll.** Set aside after review: the health page would wait on the disk under observation.
* **A separate vitals poll, or writing samples into the state-history log.** Set aside after review: the history file is for up and down transitions, and a second poll is extra load on the same page.
* **One platform row, and a disk meter on every platform workload.** Set aside after review: the controller, admin UI, and Kong have separate resource use, and Kong and the admin UI have no data volume. Host directories used for config are not that volume.
* **A weighted score across the three meters.** Set aside after review: the weights are a policy the page does not need. The scarce machine resource is the order.

## Key Decisions & Staging for Next Steps

* The denominator for every meter is the machine: memory, disk, and processor. A volume request is a label. A real cap, such as Redis `maxmemory`, is a mark, not the rank.
* The top row is the workload holding the most of the resource with the least free space. Stopped workloads compete on retained disk only. Unknown sorts above numeric rows. Ties use the component key.
* Processor share is the change since the previous in-memory counter. The first interval after a restart shows no processor figure.
* Disk is sampled off the overview request, one walk at a time, slower than the page poll. The response shows the last finished disk sample and its time. Memory and processor can be newer.
* Vitals are an optional field on `GET /api/v1/overview`. Components, summary, and history are unchanged. No second endpoint.
* Rows are each data-service instance, plus controller, admin UI, and Kong. Several volumes are listed; their bytes add for the disk share. Host config directories are excluded.
* The drawing is one row of meters with used, machine total, and percent printed, and the deciding meter named. The hour-range control does not apply. Reuse the existing progress bar and byte formatting.
* Do not change charts, the spec, or volumes to take a reading. Do not append samples to the state-history log.

All seven review lenses completed: design, codebase details, zoom-out, comprehension, implementation readiness, structural, and inventive catalyst.
