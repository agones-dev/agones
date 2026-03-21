---
title: "Documentation Editing and Contribution"
linkTitle: "Documentation Editing and Contribution"
weight: 1000
description: >
  How to contribute to the docs
---

## Writing Documentation

We welcome contributions to documentation! 

## Running the site

If you clone the repository and run `make site-server` from the `build` directory, this will run the live hugo server, 
running as a showing the next version's content and upcoming `publishedDate`'d content as well. This is likely to be
the next releases' version of the website.

To show the current website's view, run it as `make site-server ENV=RELEASE_VERSION={{< release-version >}} ARGS=`

## Platform

This site and documentation is built with a combination of Hugo, static site generator,
with the Docsy theme for open source documentation.

- [Hugo Documentation](https://gohugo.io/documentation/)
- [Docsy Guide](https://github.com/google/docsy)
- [Link Checker](https://github.com/wjdp/htmltest)

## Documentation for upcoming features

Since we may want to update the existing documentation for a given release, when writing documentation,
you will likely want to make sure it doesn't get published until the next release.

There are two approaches for hiding documentation for upcoming features, such that it only gets published when the
version is incremented on the release:

### At the Page Level

Use `publishDate` and `expiryDate` in [Hugo Front Matter](https://gohugo.io/content-management/front-matter/) to control
when the page is displayed (or hidden). You can look at the [Release Calendar](https://github.com/agones-dev/agones/blob/main/docs/governance/release_process.md#release-calendar)
to align it with the next release
candidate release date - or whichever release is most relevant.  

### Within a Page

We have a `feature` shortcode that can be used to show, or hide sections of pages based on the current semantic version
(set in config.toml, and overwritable by env variable).

For example, to show a section only from 1.24.0 forwards:

```markdown
{{%/* feature publishVersion="1.24.0" */%}}
  This is my special content that should only display >= 1.24.0
{{%/* /feature */%}}
```

or to hide a section from 1.24.0 onward:

```markdown
{{%/* feature expiryVersion="1.24.0" */%}}
  This is my special content that will be hidden >= 1.24.0
{{%/* /feature */%}}
```

## Regenerate Diagrams

To regenerate the [PlantUML](http://plantuml.com/) or [Dot](https://www.graphviz.org/doc/info/lang.html) diagrams,
delete the .png version of the file in question from `/site/static/diagrams/`, and run `make site-images`.