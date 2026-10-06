// {{ $faviconDarkExists := fileExists (path.Join "static" "favicon-dark.svg") }}
// {{/* fileExists sees the project's static/ folder only, not files mounted onto static (module mounts): params.favicon.dark says the dark favicon is there. */}}
// {{ with site.Params.favicon }}{{ if .dark }}{{ $faviconDarkExists = true }}{{ end }}{{ end }}
(function () {
  const faviconEl = document.getElementById("favicon-svg");
  const faviconDarkExists = "{{ $faviconDarkExists }}" === "true";

  if (faviconEl && faviconDarkExists) {
    const lightFavicon = '{{ "favicon.svg" | relURL }}';
    const darkFavicon = '{{ "favicon-dark.svg" | relURL }}';

    const darkModeQuery = window.matchMedia("(prefers-color-scheme: dark)");

    function updateFavicon(e) {
      faviconEl.href = e.matches ? darkFavicon : lightFavicon;
    }

    // Set favicon on load
    updateFavicon(darkModeQuery);

    // Listen for system preference changes
    darkModeQuery.addEventListener("change", updateFavicon);
  }
})();
