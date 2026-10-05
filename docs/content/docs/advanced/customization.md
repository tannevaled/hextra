---
title: Customizing Hextra
linkTitle: Customization
---

Hextra offers some default customization options in the `hugo.yaml` config file to configure the theme.
This page describes the available options and how to customize the theme further.

<!--more-->

## Custom CSS

To add custom CSS, we need to create a file `assets/css/custom.css` in our site. Hextra will automatically load this file.

### Font Family

The font family of the content can be customized using:

```css {filename="assets/css/custom.css"}
.content {
  font-family: "Times New Roman", Times, serif;
}
```

### Inline Code Element

The color of text mixed with `other text` can customized with:

```css {filename="assets/css/custom.css"}
.content code:not(.code-block code) {
  color: #c97c2e;
}
```

### Primary Color

The primary color of the theme can be customized by setting the `--primary-hue`, `--primary-saturation` and `--primary-lightness` variables:

```css {filename="assets/css/custom.css"}
:root {
  --primary-hue: 100deg;
  --primary-saturation: 90%;
  --primary-lightness: 50%;
}
```

### Component Layout Variables

Hextra provides CSS variables to customize the width of pages, navbar, and footer:

```css {filename="assets/css/custom.css"}
:root {
  /* Page width - also configurable via hugo.yaml params.page.width */
  --hextra-max-page-width: 80rem; /* default: 80rem (normal), 90rem (wide), 100% (full) */

  /* Navbar width - also configurable via hugo.yaml params.navbar.width */
  --hextra-max-navbar-width: 90rem; /* independent navbar width */

  /* Footer width - also configurable via hugo.yaml params.footer.width */
  --hextra-max-footer-width: 80rem; /* independent footer width */
}
```

### Tailwind Theme Variables

Starting with Hextra v0.10.0, which is built on Tailwind CSS v4, you can customize the theme by overriding CSS variables inside the `@layer theme` block.

This lets you customize the global look and feel without having to modify every individual class.

```css {filename="assets/css/custom.css"}
@layer theme {
  :root {
    --hx-default-mono-font-family: "JetBrains Mono", monospace;
  }
}
```

Check out [Tailwind Theme Variables documentation](https://tailwindcss.com/docs/theme#default-theme-variable-reference) for details.

### Further Theme Customization

The theme can be further customized by overriding the default styles via the exposed css classes. An example for customizing the footer element:

```css {filename="assets/css/custom.css"}
.hextra-footer {
  /* Styles will be applied to the footer element */
}

.hextra-footer:is(html[class~="dark"] *) {
  /* Styles will be applied to the footer element in dark mode */
}
```

The following classes can be used to customize various parts of the theme.

#### General

- `hextra-scrollbar` - The scrollbar element
- `content` - Page content container

#### Shortcodes

##### Badge

- `hextra-badge` - The badge element

##### Card

- `hextra-card` - The card element
- `hextra-card-image` - The card image element
- `hextra-card-icon` - The card icon element
- `hextra-card-subtitle` - The card subtitle element

##### Cards

- `hextra-cards` - The cards grid container

##### Jupyter Notebook

- `hextra-jupyter-code-cell` - The Jupyter code cell container
- `hextra-jupyter-code-cell-outputs-container` - The Jupyter code cell outputs container
- `hextra-jupyter-code-cell-outputs` - The Jupyter code cell output div element

##### PDF

- `hextra-pdf` - The PDF container element

##### Steps

- `hextra-steps` - The steps container

##### Tabs

- `hextra-tabs-panel` - The tabs panel container
- `hextra-tabs-toggle` - The tabs toggle button

##### Filetree

- `hextra-filetree` - The filetree container

##### Folder

- `hextra-filetree-folder` - The filetree folder container

#### Navbar

- `hextra-nav-container` - The navbar container
- `hextra-nav-container-blur` - The navbar container in blur element
- `hextra-hamburger-menu` - The hamburger menu button

#### Footer

- `hextra-footer` - The footer element
- `hextra-custom-footer` - The custom footer section container

#### Search

- `hextra-search-trigger` - The search trigger button
- `hextra-search-dialog` - The search dialog element
- `hextra-search-input` - The search input element
- `hextra-search-results` - The search results list container

Optional nested classes and selectors used within the search UI:

- `hextra-search-crumb` - The breadcrumb label for the first result from a page
- `hextra-search-title` - The result title element
- `hextra-search-excerpt` - The result snippet text
- `hextra-search-match` - The highlighted query span
- `hextra-search-empty` - The empty state element
- `a[role="option"][aria-selected="true"]` - The selected result anchor

#### Table of Contents

- `hextra-toc` - The table of contents container

#### Sidebar

- `hextra-sidebar-container` - The sidebar container
- `hextra-sidebar-active-item` - The active item in the sidebar

#### Language Switcher

- `hextra-language-switcher` - The language switcher button
- `hextra-language-options` - The language options container

#### Theme Toggle

- `hextra-theme-toggle` - The theme toggle button

#### Code Copy Button

- `hextra-code-copy-btn-container` - The code copy button container
- `hextra-code-copy-btn` - The code copy button
- `hextra-copy-icon` - The copy icon element
- `hextra-success-icon` - The success icon element

#### Code Block

- `hextra-code-block` - The code block container
- `hextra-code-filename` - The filename element for code blocks

#### Feature Card

- `hextra-feature-card` - The feature card link element

#### Feature Grid

- `hextra-feature-grid` - The feature grid container

### Syntax Highlighting

List of available syntax highlighting themes are available at [Chroma Styles Gallery](https://xyproto.github.io/splash/docs/all.html). The stylesheet can be generated using the command:

```shell
hugo gen chromastyles --style=github
```

To override the default syntax highlighting theme, we can add the generated styles to the custom CSS file.

## Custom Scripts

You may add custom scripts to the end of the head for every page by adding the following file:

```
layouts/_partials/custom/head-end.html
```

## Custom Extra Section in Footer

You can add extra section in the footer by creating a file `layouts/_partials/custom/footer.html` in your site.

```html {filename="layouts/_partials/custom/footer.html"}
<!-- Your footer element here -->
```

The added section will be added before the copyright section in the footer.
You can use [HTML](https://developer.mozilla.org/en-US/docs/Web/HTML) and [Hugo template syntax](https://gohugo.io/templates/) to add your own content.

Hugo variables available in the footer section are: `.switchesVisible` and `.displayCopyright`.

## Custom Page Sections

You can add custom sections around each page's content by creating any of these files in your site:

```
layouts/_partials/custom/page-begin.html
layouts/_partials/custom/content-begin.html
layouts/_partials/custom/content-end.html
layouts/_partials/custom/page-end.html
```

The page hooks are rendered inside the page's `<main>` element. The content hooks are rendered immediately before and after the page content.

Each partial receives the current Hugo page as context, so you can use page parameters, site parameters, and other Hugo template features.

## Custom Navbar Items

You can add elements to the navbar, right before the search field, by creating a file `layouts/_partials/custom/navbar-before-search.html` in your site.
It receives the current page as context, so it can for instance render a version selector that points at the current page in another version.
It is only rendered when the main menu has a search item.

## Custom Sidebar Root

By default, the sidebar lists the tree of the current page's first section.
To root it elsewhere, create `layouts/_partials/custom/sidebar-root.html` in your site and return the section page to use, or `false` to keep the default.
It receives the current page as context.

```html {filename="layouts/_partials/custom/sidebar-root.html"}
{{- /* Root the sidebar at /docs/<version>, so other versions are not listed. */ -}}
{{- $root := false -}}
{{- with site.GetPage "/docs" -}}
  {{- range .Sections -}}
    {{- if or (eq $.Path .Path) (strings.HasPrefix $.Path (printf "%s/" .Path)) -}}
      {{- $root = . -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- return $root -}}
```

When a custom root is returned, the mobile sidebar shows that tree as well, instead of the main menu entries.

## Custom Search Scope

To restrict search results to part of the site, create `layouts/_partials/custom/search-scope.html` in your site and return a section page: only the pages under it are searched. Return `false` to search the whole site, which is the default.
It receives the current page as context, so a versioned documentation can search only the version being read:

```html {filename="layouts/_partials/custom/search-scope.html"}
{{- /* Search only the version being read, rooted like the sidebar. */ -}}
{{- return partial "custom/sidebar-root.html" . -}}
```

The restriction is applied inside the index, so the result limit (`maxPageResults`) counts only pages in scope.
## Custom Switches

To add items next to the theme toggle (and the language switch), create `layouts/_partials/custom/switches.html` in your site.
They are rendered at the start of the switches row — the bottom of the sidebar, or the footer on pages without a sidebar — before the language switch. The partial receives the current page as context.

```html {filename="layouts/_partials/custom/switches.html"}
<a href="https://example.org" title="Example" class="hx:p-2" target="_blank" rel="noreferrer">
  {{- partial "utils/icon.html" (dict "name" "globe-alt" "attributes" "height=16") -}}
</a>
```

## Related Sites From the Logo

`params.navbar.logo.menu` lists related sites. A chevron then appears beside the navbar title and opens them as a dropdown, while the logo itself still leads home. Each entry has a `name`, a `url`, and optionally a `logo` and a `logoDark` image; links leaving the site open in a new tab with an up-right arrow. An entry `separator: true` draws a line between two groups. The chevron points forward while the menu is closed and turns down when it opens.

```yaml {filename="hugo.yaml"}
params:
  navbar:
    logo:
      path: images/logo.svg
      menu:
        - name: Sister project
          url: https://sister.example.org/
          logo: images/sister.png
          logoDark: images/sister-dark.png
```

On narrow screens the navbar combos show their icon and chevron only, and below 414px the title gives way to the logo, so the brand and its menu never leave the screen.

## Navbar Items From a Partial

A main menu item of type `partial` renders a partial of your site at its place in the navbar, with the current page as context. Its `weight` orders it among the other items, so it can sit anywhere — for instance a version selector between other controls:

```yaml {filename="hugo.yaml"}
menu:
  main:
    - name: Version
      weight: 3
      params:
        type: partial
        partial: custom/version-select.html
```

Such items are left out of the mobile sidebar's menu, like the search and the switches.

## Theme Toggle as a Navbar Combo

The theme toggle can sit in the navbar as a combo — the icon and the name of the current choice, and a chevron — drawn in the same box as the search field:

```yaml {filename="hugo.yaml"}
menu:
  main:
    - name: Theme
      weight: 4
      params:
        type: theme-toggle
        combo: true
params:
  theme:
    displayToggle: false  # and no longer at the foot of the sidebar
```

The box is the `hextra-navbar-combo` class, which a site's own navbar dropdown can take to look the same.

## Forge Icons

The icon set includes `github`, `gitlab`, `codeberg`, `forgejo` and `git`, so a navbar link to the source repository can show the right forge:

```yaml {filename="hugo.yaml"}
menu:
  main:
    - name: Sources
      url: https://gitlab.example.org/group/project
      params:
        icon: gitlab
```

## External Link Badge

Icon-only navbar links that leave the site get a small up-right arrow on their top-right corner. To mark your own icon links the same way (for instance in `custom/switches.html`), give the link the `hextra-icon-link` class and render the badge inside it:

```html
<a class="hextra-icon-link" href="https://example.org" target="_blank" rel="noreferrer">
  <img src="/images/example.png" alt="Example" width="20" height="20">
  {{- partial "utils/external-badge.html" . -}}
</a>
```

## Language Flags

Give each language a `flag` parameter (an image path, relative to `static/`) to show it in the language switch, on the button (after the switch's icon) and on every option:

```yaml {filename="hugo.yaml"}
languages:
  en:
    label: English
    params:
      flag: images/flags/gb.svg
```

## Page Metadata

Four options, all off by default, use a page's front matter `description` and `tags`:

```yaml {filename="hugo.yaml"}
params:
  page:
    displayDescription: true  # the description as a subtitle under a docs page title
    displayReadingTime: true  # "3 min read" under it
    sectionCards: true        # on a docs section page, a card per child page or section
  toc:
    displayTags: true         # the page's tags at the bottom of the right sidebar
    displayRelated: true      # then the pages sharing the most tags with it
    relatedLimit: 5
```

Section cards show each child's title and description, ordered by weight; a page opts out with `sectionCards: false` in its front matter. Related pages are counted within the current language.

## Search Engine Verification

To prove ownership of the site to a search engine, give its code:

```yaml {filename="hugo.yaml"}
params:
  seo:
    verification:
      google: "…"   # google-site-verification
      bing: "…"     # msvalidate.01
      yandex: "…"   # yandex-verification
```

## Language Switch Icon

The language switch shows a globe by default. Set `params.languageSwitch.icon` to another icon name of the theme — `translate`, for instance — to use it wherever the switch appears:

```yaml {filename="hugo.yaml"}
params:
  languageSwitch:
    icon: translate
```

## Custom Layouts

The layouts of the theme can be overridden by creating a file with the same name in the `layouts` directory of your site.
For example, to override the `single.html` layout for docs, create a file `layouts/docs/single.html` in your site.

For further information, refer to the [Hugo Templates][hugo-template-docs].

## Further Customization

Didn't find what you were looking for? Feel free to [open a discussion](https://github.com/imfing/hextra/discussions) or make a contribution to the theme!

[hugo-template-docs]: https://gohugo.io/templates/
