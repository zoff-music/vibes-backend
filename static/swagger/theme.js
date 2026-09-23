(() => {
  const root = document.documentElement;

  root.dataset.theme = window.matchMedia("(prefers-color-scheme: light)")
    .matches
    ? "light"
    : "dark";

  document.getElementById("theme").addEventListener("click", () => {
    root.dataset.theme = root.dataset.theme === "dark" ? "light" : "dark";
  });

  let ui;
  let adminVisible = false;
  let loading = false;

  const access = document.getElementById("access");
  const specLink = document.getElementById("spec-link");

  async function refresh() {
    if (loading) {
      return;
    }

    loading = true;
    document.getElementById("docs-error").hidden = true;

    try {
      const adminResponse = await fetch("/api/swagger/admin.json", {
        credentials: "same-origin",
        cache: "no-store",
      });

      const isAdmin = adminResponse.ok;

      if (!isAdmin && adminVisible) {
        document.getElementById("swagger-ui").replaceChildren();
        ui = null;
        adminVisible = false;
        access.textContent = "Public reference";
        specLink.href = "/api/swagger/doc.json";
      }

      const response = isAdmin
        ? adminResponse
        : await fetch("/api/swagger/doc.json", {
            credentials: "same-origin",
            cache: "no-store",
          });

      if (!response.ok) {
        throw new Error("Unable to load documentation");
      }

      const spec = await response.json();

      if (!ui || isAdmin !== adminVisible) {
        ui = SwaggerUIBundle({
          spec,
          dom_id: "#swagger-ui",
          deepLinking: true,
          docExpansion: "list",
          filter: true,
          tagsSorter: "alpha",
          operationsSorter: "alpha",
          displayRequestDuration: true,
          defaultModelsExpandDepth: 1,
          validatorUrl: null,
          presets: [SwaggerUIBundle.presets.apis],
          layout: "BaseLayout",
          requestInterceptor: (request) => {
            request.credentials = "same-origin";

            return request;
          },
          responseInterceptor: (response) => {
            if (/\/api\/v1\/admin\/sessions(?:\?|$)/.test(response.url)) {
              window.setTimeout(refresh, 0);
            }

            return response;
          },
        });
      }

      adminVisible = isAdmin;
      access.textContent = isAdmin
        ? "Admin reference · authenticated"
        : "Public reference";
      specLink.href = isAdmin
        ? "/api/swagger/admin.json"
        : "/api/swagger/doc.json";
    } catch (error) {
      // A failed session check must never leave an admin specification on screen.
      document.getElementById("swagger-ui").replaceChildren();
      ui = null;
      adminVisible = false;
      access.textContent = "Reference unavailable";
      specLink.href = "/api/swagger/doc.json";
      document.getElementById("docs-error").hidden = false;
    } finally {
      loading = false;
    }
  }

  document.getElementById("refresh").addEventListener("click", refresh);
  window.addEventListener("focus", refresh);
  window.addEventListener("pageshow", (event) => {
    if (event.persisted) {
      refresh();
    }
  });

  window.setInterval(() => {
    if (!document.hidden) {
      refresh();
    }
  }, 60000);

  refresh();
})();
