import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const environment = loadEnv(mode, process.cwd(), "");
  const allowedHost = environment.LYRA_DEV_ALLOWED_HOST;
  return {
    plugins: [react()],
    server: { port: 5173, allowedHosts: allowedHost ? [allowedHost] : undefined },
  };
});
