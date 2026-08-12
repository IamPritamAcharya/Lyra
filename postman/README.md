# Lyra Postman collection

Import both JSON files into Postman. Select the **Lyra Local** environment and update the two audio paths if your files live elsewhere.

Run requests in this order: Health / Ready, Admin tracks / Create track, Upload reference audio, Get track until `Status` is `READY`, then Identification / Identify indexed query clip. The Create request automatically saves its response `PublicID` as `track_id`.
