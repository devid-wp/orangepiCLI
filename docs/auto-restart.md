# Automatic restart policy

`auto_restart` uses a bounded recovery policy: at most **3** retries, with a
**5-second** delay between attempts. A successful restart ends the recovery
loop; exhausting the budget leaves the slot stopped. This avoids an infinite
crash/restart cycle and will be driven by the long-lived supervisor added with
process monitoring.
