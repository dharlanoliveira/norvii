import react from "@vitejs/plugin-react";
import { defineConfig, type ProxyOptions } from "vite";

const defaultAPIOrigin = "http://127.0.0.1:8080";

const maintainerEvaluationRouteContexts = [
  "^/api/v1/evaluations(?:/|\\?|$)",
  "^/api/v1/evaluation-datasets(?:/|\\?|$)",
] as const;

export function maintainerEvaluationProxy(
  token: string | undefined,
  target = process.env.NORVII_API_ORIGIN ?? defaultAPIOrigin,
): ProxyOptions {
  const headers =
    token === undefined || token === ""
      ? undefined
      : { Authorization: `Bearer ${token}` };

  return {
    target,
    changeOrigin: true,
    ...(headers === undefined ? {} : { headers }),
  };
}

export function maintainerEvaluationProxies(
  token: string | undefined,
  target = process.env.NORVII_API_ORIGIN ?? defaultAPIOrigin,
): Record<string, ProxyOptions> {
  const proxy = maintainerEvaluationProxy(token, target);

  return Object.fromEntries(
    maintainerEvaluationRouteContexts.map((context) => [context, proxy]),
  );
}

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      ...maintainerEvaluationProxies(
        process.env.NORVII_EVALUATION_MAINTAINER_TOKEN,
      ),
      "/api": defaultAPIOrigin,
    },
  },
  build: {
    manifest: true,
  },
});
