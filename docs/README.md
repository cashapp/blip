# Blip Docs

This directory contains the documentation source for [Blip](https://github.com/cashapp/blip].
Docs are written in Markdown, statically generated into HTML, and served by GitHub Pages.

Run locally:

```sh
bundle exec jekyll serve --incremental
```

To build:

```sh
rm -rf _site/*
bundle exec jekyll build
```
