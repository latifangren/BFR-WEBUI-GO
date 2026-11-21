package handlers

import (
	"net/http"
)

const apiDocsHTML = `<!doctype html>
<html lang="en">
<head>
  <title>BFR API Reference</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    :root { color-scheme: dark; font: 16px/1.5 system-ui, sans-serif; background: #0f172a; color: #e2e8f0; }
    body { max-width: 960px; margin: 0 auto; padding: 24px 16px; }
    header { display: flex; justify-content: space-between; align-items: baseline; gap: 16px; flex-wrap: wrap; }
    a { color: #7dd3fc; }
    section { margin: 18px 0; padding: 16px; border: 1px solid #334155; border-radius: 8px; }
    article { margin: 12px 0; padding: 12px; background: #1e293b; border-radius: 6px; }
    code { overflow-wrap: anywhere; }
    .method { display: inline-block; min-width: 4.5em; margin-right: 10px; color: #86efac; font-weight: 700; }
    .muted { color: #94a3b8; }
  </style>
</head>
<body>
  <header><h1>BFR API Reference</h1><a href="/openapi.json">OpenAPI JSON</a></header>
  <p class="muted">Rendered locally from the embedded OpenAPI specification.</p>
  <main id="api-reference">Loading API specification…</main>
  <script type="module">
    const root = document.getElementById('api-reference')
    try {
      const response = await fetch('/openapi.json')
      if (!response.ok) throw new Error('HTTP ' + response.status)
      const spec = await response.json()
      const paths = Object.entries(spec.paths || {})
      root.replaceChildren()
      for (const [path, operations] of paths) {
        const section = document.createElement('section')
        const heading = document.createElement('h2')
        heading.textContent = path
        section.append(heading)
        for (const [method, operation] of Object.entries(operations || {})) {
          const article = document.createElement('article')
          const title = document.createElement('div')
          const verb = document.createElement('span')
          verb.className = 'method'
          verb.textContent = method.toUpperCase()
          const summary = document.createElement('span')
          summary.textContent = operation.summary || operation.operationId || 'Endpoint'
          title.append(verb, summary)
          article.append(title)
          if (operation.description) {
            const description = document.createElement('p')
            description.textContent = operation.description
            article.append(description)
          }
          section.append(article)
        }
        root.append(section)
      }
      if (paths.length === 0) root.textContent = 'No API paths are defined.'
    } catch (error) {
      root.textContent = 'Could not load the embedded API specification: ' + String(error)
    }
  </script>
</body>
</html>`

func HandleAPIDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(apiDocsHTML))
}
