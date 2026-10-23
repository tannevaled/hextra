---
title: "Multi-language"
weight: 1
prev: /docs/advanced/additional-pages
---

Hextra supports creating site with multiple languages using Hugo's [multilingual mode](https://gohugo.io/content-management/multilingual/).

<!--more-->

## Enable Multi-language

To make our site multi-language, we need to tell Hugo the supported languages. We need to add to the site configuration file:

```yaml {filename="hugo.yaml"}
defaultContentLanguage: en
languages:
  en:
    label: English
    weight: 1
  fr:
    label: Français
    weight: 2
  ja:
    label: 日本語
    weight: 3
```

> [!NOTE]
> Older Hugo configurations may use `languageName`, `languageCode`, and `languageDirection`.
> Starting with Hugo v0.158.0, use `label`, `locale`, and `direction` respectively.
> See Hugo's [language settings documentation](https://gohugo.io/configuration/languages/#language-settings).

## Manage Translations by Filename

Hugo supports managing translations by filename. For example, if we have a file `content/docs/_index.md` in English, we can create a file `content/docs/_index.fr.md` for French translation.

{{< filetree/container >}}
  {{< filetree/folder name="content" >}}
    {{< filetree/folder name="docs" state="open" >}}
      {{< filetree/file name="_index.md" >}}
      {{< filetree/file name="_index.fr.md" >}}
      {{< filetree/file name="_index.ja.md" >}}
    {{< /filetree/folder >}}
  {{< /filetree/folder >}}
{{< /filetree/container >}}

Note: Hugo also supports [Translation by content directory](https://gohugo.io/content-management/multilingual/#translation-by-content-directory).

## Translate Menu Items

To translate menu items in the navigation bar, we need to set the `identifier` field:

```yaml {filename="hugo.yaml"}
menu:
  main:
    - identifier: documentation
      name: Documentation
      pageRef: /docs
      weight: 1
    - identifier: blog
      name: Blog
      pageRef: /blog
      weight: 2
```

and translate them in the corresponding i18n file:

```yaml {filename="i18n/fr.yaml"}
documentation: Documentation
blog: Blog
```

## Translate Strings

To translate strings on the other places, we need to add the translation to the corresponding i18n file:

```yaml {filename="i18n/fr.yaml"}
readMore: Lire la suite
```

A list of strings used in the theme can be found in the `i18n/en.yaml` file.

## Language Flags

Give each language a `flag` parameter to show it in the language switch, on the button (after the switch's icon) and on every option. The value is a flag code:

```yaml {filename="hugo.yaml"}
languages:
  en:
    label: English
    params:
      flag: gb
  fr:
    label: Français
    params:
      flag: fr
```

Every flag of [flag-icons](https://flagicons.lipis.dev/) (MIT) can be used: the 249 [ISO 3166-1 alpha-2](https://www.iso.org/iso-3166-country-codes.html) codes (`fr`, `de`, `jp`…) and 22 others, such as `eu`, `un`, `gb-sct` (Scotland), `gb-wls` (Wales), `es-ct` (Catalonia), `es-pv` (Basque Country) or `xk` (Kosovo).
The flag is fetched once at build time, cached by Hugo, and published with the site, so visitors' browsers never load it from a CDN.

- To provide your own image for a code, or to build offline, put it in your site's `assets/flags/<code>.svg`: it is used first.
- To fetch flags from elsewhere (a mirror, another version), set the URL template, where `%s` is the code; set `enable: false` to use only `assets/flags/`:

  ```yaml {filename="hugo.yaml"}
  params:
    flags:
      remote:
        enable: true
        url: https://cdn.jsdelivr.net/npm/flag-icons@7.5.0/flags/4x3/%s.svg
  ```

A value containing `/` or `.` is an image path relative to `static/` (or a URL) and is used as is, e.g. `flag: images/flags/gb.svg`.

### Language Switch Icon

The language switch shows a globe by default, also next to a flag: the flag says which language, the icon says that this changes it. Set `params.languageSwitch.icon` to another icon of the theme, `translate` for instance, to use it in the sidebar and the footer. A `language-switch` item of the navbar menu keeps its own `icon` (`translate` by default):

```yaml {filename="hugo.yaml"}
params:
  languageSwitch:
    icon: translate
```

## Read More

- [Hugo Multilingual Mode](https://gohugo.io/content-management/multilingual/)
- [Hugo Multilingual Part 1: Content translation](https://www.regisphilibert.com/blog/2018/08/hugo-multilingual-part-1-managing-content-translation/)
- [Hugo Multilingual Part 2: Strings localization](https://www.regisphilibert.com/blog/2018/08/hugo-multilingual-part-2-i18n-string-localization/)
