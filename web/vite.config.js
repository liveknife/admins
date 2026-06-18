import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig(function (_a) {
    var mode = _a.mode;
    var env = loadEnv(mode, process.cwd(), "");
    return {
        plugins: [react()],
        server: {
            proxy: {
                "/api": {
                    target: env.VITE_API_PROXY_TARGET || "http://localhost:8080",
                    changeOrigin: true
                },
                "/uploads": {
                    target: env.VITE_API_PROXY_TARGET || "http://localhost:8080",
                    changeOrigin: true
                }
            }
        }
    };
});
