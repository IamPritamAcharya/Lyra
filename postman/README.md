# Lyra Postman collection

Import both JSON files into Postman. Select the **Lyra Local** environment and update the two audio paths if your files live elsewhere.

Run requests in this order: Health / Ready, Admin tracks / Create track, Upload reference audio, Get track until `Status` is `READY`, then Identification / Identify indexed query clip. The Create request automatically saves its response `PublicID` as `track_id`.
# Lyra Postman collection

Import the collection and the local environment. Set `admin_password` in the selected environment to the plain-text password that matches `LYRA_ADMIN_PASSWORD_HASH` in the running API configuration. Do not put the bcrypt hash in Postman.

Run **Admin auth / Login (stores CSRF token)** first. Postman retains the HttpOnly session cookie in its cookie jar and the test script stores the returned CSRF token for catalog changes. Then create a track, upload its reference audio, poll until its `Status` is `READY`, and send an identification query.
