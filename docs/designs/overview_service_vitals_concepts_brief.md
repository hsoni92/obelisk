# Overview service vitals

* **Author:** hsoni92
* **Date:** 2026-11-16
* **Revised:** 2026-11-16 (fixed-size in-memory rings; the page does not collect)

## Overview

Obelisk runs many workloads on one machine. The Overview page shows whether those workloads are up, and it shows backup status. It does not show which workload is using the memory, processor, or disk the machine is short of.

This feature adds a vitals block on that page. Each data-service instance is one row. The platform is three rows: the controller, the admin UI, and Kong. Each row has filled meters for memory, processor, and disk, plus a short line of the last half hour so a climb is visible. The fill is that workload's share of the machine. The row at the top holds the most of whichever of those three the machine is shortest on. A stopped service stays in the list because disabling it keeps its disk.

The numbers come from fixed-size rings in the controller's memory. A background sampler writes a slot about once a minute and overwrites the oldest slot. The dashboard reads those slots. It does not measure anything when the page refreshes. A reading can be a minute old, and the disk figure can be several minutes old. That staleness is accepted so the page stays quick and the disk is not walked on every refresh.

## Analysis

Seven reviews read the first proposal: meter each service against its own memory cap and its own volume size, sort by whichever meter is fullest, and keep only the latest sample. The caps are not enforced on this appliance. The author then required the samples to sit in fixed-size circular buffers, to stay off the disk, and to stay off the dashboard request, accepting a stale reading in exchange.

### The limits in the proposal are not enforced

`hardware.tier` is the word `floor` or `comfort`. Nothing applies it. No container sets a memory limit. The only memory ceiling in the charts is Redis `--maxmemory 256mb`, which makes Redis drop keys. It is not the kernel limit that kills a process, and it does not change with the tier. A volume's requested size (Postgres 10Gi, Redis 2Gi, RustFS 20Gi, and the others) is a claim. On this node's local disk layout that claim is not a quota. A volume can sit at a small fraction of its claim while the machine disk is full, and the sum of the claims can be larger than the disk. k3s, container images, and the controller's own directories sit outside every service row.

Sorting by "fullest of three unlike meters" then promotes the wrong row. A Redis process near 256 MB, or a small volume near its claim, ranks above a Postgres process that is using most of the machine's RAM.

The reviews split on what to do about the missing caps. One side would build real per-service memory limits and meter against those. The other side treats that as a second project: writing limits restarts pods and can turn on Redis eviction. This brief takes the second side. The meters use the machine as the denominator. Where a real cap already exists, it can be drawn as a mark. This feature does not create caps.

### What each meter means

Memory is the workload's use at the newest slot, divided by the machine's memory. Processor is that workload's share of the machine across the gap between the two newest slots. Disk is bytes held by that workload's volumes, divided by the machine's disk. The volume's requested size is labeled as a request beside the meter.

Processor share needs two slots. A single counter is a total, not a rate. The ring supplies the previous counter. After a restart the first slot has no processor figure. Showing zero there would rank a busy service as idle.

### Rings, and why the page does not collect

The overview handler already loads health, backup readiness, and the timeline, and the page polls it about every 8 seconds. Measuring on that poll would make the health verdict wait on cgroup reads and on disk, and it would repeat that work for every open tab.

Each workload has one ring of 30 slots in process memory. Each slot holds the time, memory bytes, the processor counter, the processor share once two slots exist, and disk bytes. The machine totals have their own ring of the same length. A sampler writes one slot about every 60 seconds and then overwrites the oldest. Thirty slots is about half an hour. That is long enough to see a workload filling the box, and short enough that the whole set of rings stays a few kilobytes.

Disk bytes are refreshed on a slower clock, about every five minutes, and only one volume at a time. The sampler reads the filesystem's used-bytes for that volume. It does not list every object in a large store. Between disk refreshes, new slots copy the last disk figure and keep that figure's own time. Stopped services are included, because disable scales the workload to zero and leaves the volume. One collection runs at a time. A collection that overruns its slot is skipped, not piled onto the next one.

The overview response copies the rings out of memory. The handler never starts a measurement. If the sampler is busy, the page still returns health, backup, and the timeline with the slots it already has. The panel prints the age of the newest slot. When that age is past three minutes, the panel is marked stale. A disk figure older than the slot shows its own age.

Nothing is written to the state-history log, and the rings are not flushed to a file. That log records up and down transitions. Putting gauges in it would turn it into a metrics series and can punch gaps in the timeline. A controller restart clears the rings. The panel shows that it is collecting until the first slot exists. No size query is sent to Postgres or Redis. No metrics-server pod is added.

A failed or skipped reading is unknown. Unknown is never drawn as zero, and it does not replace the rest of the overview. Reconcile-off keeps returning components and history as it does now. While reconcile is off, the sampler does not run, and the rings stay as they were until the process restarts.

### Rows

Overview today has one platform row and one row per data-service instance. The platform release is three workloads with separate resource use: controller, admin UI, and Kong. OpenRouter and mail run inside the controller process, so they are not rows. SearXNG has two volumes (a small config claim and a data claim). The disk meter lists each volume. The row's disk share uses the sum of bytes held. The controller's host directories (`/etc/obelisk`, `/var/lib/obelisk`, `/opt/obelisk`) are appliance files, not a data volume, and they stay off the disk meter.

The rows will not add up to the machine. k3s, containerd, and backup jobs sit outside them. The block shows machine memory, processor, and disk, and the part the rows do not account for. Backup jobs are a known memory spike on the small tier and are not their own row in this feature. A spike that starts and ends inside one minute is flattened into that slot. The newest slot also carries restart count and the last stop reason (out of memory, evicted) so a crash between slots is still visible. Node pressure (memory, disk, process-id) is shown with the machine totals when the cluster reports it.

### How the list is ordered and drawn

Order uses the newest slot: the workload holding the most of the resource the machine is shortest on (least free among memory, disk, and processor). A stopped service competes on retained disk only. Its memory and processor read as not running, not as healthy zeros. A missing reading sorts as unknown, above the numeric rows, so a service that could not be measured is not hidden under the ones that look fine. Equal shares keep a stable order by the existing component key, so the list does not shuffle when nothing changed.

Each meter prints the amount used, the machine total, and the percent, and a line of that metric across the 30 slots. The meter that placed the row first is labeled in words. The section says the figures are from the ring and do not follow the hour-range control. The page keeps the last copy when a refresh fails. The existing progress bar and byte formatting are enough for the meters. The line uses the chart already on the page. There is no second range control and no way to scroll the ring back further than its 30 slots.

The vitals travel as an optional addition on the current overview response. The component list, the summary counts, and the history stay in their current order and shape. An older admin page ignores the new field. A newer page still draws health, backup, and the timeline when the field is absent.

### What this still will not catch

A processor spike shorter than the one-minute slot is flattened. A service can be saturated on one core while its share of the whole machine stays small. A service about to hit Redis's own 256 MB ceiling will not rise to the top while the machine still has free memory; the mark on that meter is the only signal. Disk can be about five minutes behind the memory and processor lines. Inodes and disk wait are left out. The failures this page is for are a full disk and a machine that is out of memory. After a controller restart the half-hour line is gone until the ring fills again.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * Memory, processor, and disk for each data-service instance and for the controller, admin UI, and Kong, on the Overview page.
  * A fixed ring of 30 slots per workload and one for the machine, kept only in controller memory, overwritten in place.
  * A sampler on its own minute clock. Disk used-bytes refresh about every five minutes, one volume at a time, without listing the volume's contents.
  * The overview request only copies the rings. It never starts a measurement.
  * Machine totals and the usage the rows do not account for.
  * Disk bytes for stopped services whose volumes are still on the machine.
  * Each of a service's volumes shown, with the row's disk share taken from the bytes they hold together.
  * Order by the largest share of the resource the machine is shortest on in the newest slot, with unknown readings visible and a stable tie-break.
  * Restart count and last stop reason on the newest slot, plus node pressure when the cluster reports it.
  * The age of the newest slot, and a stale mark when that age is past three minutes.
  * A missing reading stays unknown and leaves the rest of the page intact.
* **Out-of-Scope Non-Goals:**
  * Writing memory limits, changing Redis `maxmemory`, resizing volumes, or starting a stopped pod in order to measure it.
  * Persisting the rings to disk, appending gauges to the state-history log, or keeping more than 30 slots.
  * Prometheus, Grafana, or a metrics-server pod.
  * A second poll, a second endpoint, or a new API version.
  * Collecting metrics inside the overview handler or on each dashboard refresh.
  * Changing the component list, summary counts, or the state-history log.
  * Rows for OpenRouter, mail, backup jobs, k3s, or containerd.
  * Per-core saturation, inode counts, and disk wait.
  * Treating a volume's requested size as a quota.
  * Letting the hour-range control change the vitals ring.

## Alternatives Weighed

* **Sort by the fullest of each service's own memory cap, volume size, and processor share.** This was the anchored hypothesis. Set aside after review: those caps are not configured, and a volume request is not a quota, so the top row would not be the workload filling the machine.
* **Build tier memory profiles and enforce container limits first, then meter against them.** Set aside for this feature: it restarts pods and can change Redis eviction. It is a separate change. The meter can gain a cap mark later if those limits exist.
* **Host gauges only, with no per-service rows.** Set aside by the author: the operator needs to see which service to act on.
* **Measure on every dashboard refresh.** Set aside by the author: a stale slot is acceptable, and the refresh must not wait on collection or add disk reads.
* **Keep only the newest sample, with no ring.** Set aside by the author: the samples sit in a fixed-size circular buffer so the panel can show the last half hour, still bounded.
* **Append samples to the state-history log, or flush the ring to a file on each slot.** Set aside by the author: that strains the disk and mixes gauges into the up/down log. A restart may start from an empty ring.
* **A separate vitals poll.** Set aside after review: the page already polls overview. The rings ride along on that response.
* **One platform row, and a disk meter on every platform workload.** Set aside after review: the controller, admin UI, and Kong have separate resource use, and Kong and the admin UI have no data volume. Host directories used for config are not that volume.
* **A weighted score across the three meters.** Set aside after review: the weights are a policy the page does not need. The scarce machine resource is the order.

## Key Decisions & Staging for Next Steps

* The denominator for every meter is the machine: memory, disk, and processor. A volume request is a label. A real cap, such as Redis `maxmemory`, is a mark, not the rank.
* Samples live in fixed-size rings of 30 slots in controller memory. The sampler appends about once a minute and overwrites the oldest slot. The rings are not written to disk.
* Disk used-bytes refresh about every five minutes, one volume at a time, from the filesystem's used-bytes. Slots between refreshes reuse the last disk figure and keep its time.
* The overview handler only copies the rings. It never starts a measurement. The panel shows the age of the newest slot and marks the block stale after three minutes.
* The top row is the workload holding the most of the resource with the least free space in the newest slot. Stopped workloads compete on retained disk only. Unknown sorts above numeric rows. Ties use the component key.
* Processor share is the change between the two newest slots. The first slot after a restart shows no processor figure. A restart clears the rings.
* Vitals are an optional field on `GET /api/v1/overview`. Components, summary, and history are unchanged. No second endpoint.
* Rows are each data-service instance, plus controller, admin UI, and Kong. Several volumes are listed; their bytes add for the disk share. Host config directories are excluded.
* Each meter shows used, machine total, and percent, the deciding meter is named, and a line draws that metric across the 30 slots. The hour-range control does not apply. Reuse the existing progress bar, byte formatting, and chart.
* Do not change charts, the spec, or volumes to take a reading. Do not append samples to the state-history log.

All seven review lenses completed on the first proposal: design, codebase details, zoom-out, comprehension, implementation readiness, structural, and inventive catalyst. The ring, the minute clock, and the refusal to collect on refresh were added by the author after that review.
