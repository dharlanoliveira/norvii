import { once } from "node:events";
import { createServer, type Server } from "node:http";

import { afterEach, describe, expect, it } from "vitest";
import { createServer as createViteServer, type ViteDevServer } from "vite";

import {
  maintainerEvaluationProxies,
  maintainerEvaluationProxy,
} from "../vite.config";

const maintainerToken = "test-maintainer-token";
const servers: Array<Server | ViteDevServer> = [];

afterEach(async () => {
  await Promise.all(servers.splice(0).map(closeServer));
});

describe("maintainer evaluation proxy", () => {
  it("forwards browser inspection requests with server-side bearer authorization", async () => {
    const receivedAuthorizations = new Map<string, string | undefined>();
    const api = createServer((request, response) => {
      receivedAuthorizations.set(
        request.url ?? "",
        request.headers.authorization,
      );
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(JSON.stringify({ status: "ok" }));
    });
    servers.push(api);
    api.listen(0, "127.0.0.1");
    await once(api, "listening");
    const address = api.address();
    if (address === null || typeof address === "string") {
      throw new Error("Test API did not expose a TCP address.");
    }

    const web = await createViteServer({
      configFile: false,
      server: {
        host: "127.0.0.1",
        proxy: {
          ...maintainerEvaluationProxies(
            maintainerToken,
            `http://127.0.0.1:${String(address.port)}`,
          ),
          "/api": `http://127.0.0.1:${String(address.port)}`,
        },
      },
    });
    servers.push(web);
    await web.listen();
    const webURL = web.resolvedUrls?.local.at(0);
    if (webURL === undefined)
      throw new Error("Test web server has no local URL.");

    const evaluationResponse = await fetch(
      `${webURL}api/v1/evaluations/example`,
    );
    const datasetResponse = await fetch(
      `${webURL}api/v1/evaluation-datasets/example`,
    );

    expect(evaluationResponse.status).toBe(200);
    expect(datasetResponse.status).toBe(200);
    expect(await evaluationResponse.json()).toEqual({ status: "ok" });
    expect(await datasetResponse.json()).toEqual({ status: "ok" });
    expect(receivedAuthorizations.get("/api/v1/evaluations/example")).toBe(
      `Bearer ${maintainerToken}`,
    );
    expect(
      receivedAuthorizations.get("/api/v1/evaluation-datasets/example"),
    ).toBe(`Bearer ${maintainerToken}`);
  });

  it("does not forward the maintainer bearer to similarly prefixed routes", async () => {
    const receivedAuthorizations = new Map<string, string | undefined>();
    const api = createServer((request, response) => {
      receivedAuthorizations.set(
        request.url ?? "",
        request.headers.authorization,
      );
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(JSON.stringify({ status: "ok" }));
    });
    servers.push(api);
    api.listen(0, "127.0.0.1");
    await once(api, "listening");
    const address = api.address();
    if (address === null || typeof address === "string") {
      throw new Error("Test API did not expose a TCP address.");
    }

    const apiOrigin = `http://127.0.0.1:${String(address.port)}`;
    const web = await createViteServer({
      configFile: false,
      server: {
        host: "127.0.0.1",
        proxy: {
          ...maintainerEvaluationProxies(maintainerToken, apiOrigin),
          "/api": apiOrigin,
        },
      },
    });
    servers.push(web);
    await web.listen();
    const webURL = web.resolvedUrls?.local.at(0);
    if (webURL === undefined)
      throw new Error("Test web server has no local URL.");

    const unrelatedRoutes = [
      "/api/v1/evaluations-preview/example",
      "/api/v1/evaluation-datasets-archive/example",
    ];
    const responses = await Promise.all(
      unrelatedRoutes.map(async (route) => fetch(`${webURL}${route}`)),
    );

    expect(responses.map((response) => response.status)).toEqual([200, 200]);
    for (const route of unrelatedRoutes) {
      expect(receivedAuthorizations.get(route)).toBeUndefined();
    }
  });

  it("does not configure a browser-visible authorization header without a maintainer token", () => {
    expect(maintainerEvaluationProxy("")).not.toHaveProperty("headers");
  });
});

async function closeServer(server: Server | ViteDevServer): Promise<void> {
  if ("close" in server && "httpServer" in server) {
    await server.close();
    return;
  }
  server.close();
  await once(server, "close");
}
