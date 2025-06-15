# pdf-gen

pdf-gen is a small helper library built on top of [github.com/go-pdf/fpdf](https://github.com/go-pdf/fpdf). It allows building PDF documents using a declarative template structure. The package handles fonts and UTF-8 text so that accented characters such as `é` or `à` are rendered correctly.

## Installation

```bash
go get github.com/boomhut/pdf-gen
```

## Quick start

The package revolves around the `SpdfTemplate` type. A template is composed of pages, each page contains layers, and layers contain items. Items can be pieces of text, multiline text (`ftext`), images or QR codes.

```go
package main

import (
    "log"

    "github.com/boomhut/pdf-gen"
)

func main() {
    tpl := pdfgen.NewTemplate("demo")
    tpl.Title = pdfgen.SpdfTitle{Data: "Demo PDF"}
    tpl.Author = "Your Name"

    // one A4 page with a single layer
    tpl.AddPage(210, 297)
    tpl.AddLayer("content", pdfgen.SpdfMargin{})

    // place some text
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "Hello world!",
        Params: map[string]string{
            "font": "Helvetica",
            "size": "16",
            "x":    "10",
            "y":    "20",
        },
    })

    if err := tpl.RenderToFile("demo.pdf"); err != nil {
        log.Fatal(err)
    }
}
```

Running the program produces `demo.pdf` in the working directory.

## Fonts

`RenderToFile` expects fonts to be defined under `./resources/fonts`. The directory must contain a `fonts.json` listing the fonts that should be registered with FPDF:

```json
[
    { "Name": "Helvetica", "Path": "helvetica.json" }
]
```

Each entry refers to a font definition generated with the `makefont` utility from the FPDF distribution.

## Saving and loading templates

Templates can be saved to and loaded from JSON which can be useful to create template builders.

```go
if err := pdfgen.SaveTemplate(tpl, "template.json"); err != nil {
    return err
}
loaded, err := pdfgen.LoadTemplate("template.json")
```

`LoadTemplateFromBytes` performs the same operation from a byte slice.

## Utilities

The package exposes helpers such as:

* `CreateQR` – generate a QR code image.
* `NperPage` – split items into pages.
* `CalcResize`, `CalcResizeFromRatio` and `CalcResizeFromWidth` – resize images while keeping their aspect ratio.
* `ListFonts` – read the list of fonts from a directory.

For more usage examples see the unit tests in `template_test.go`.

## Examples

### Article PDF

```go
package main

import (
    "log"

    "github.com/boomhut/pdf-gen"
)

func main() {
    tpl := pdfgen.NewTemplate("article")
    tpl.Title = pdfgen.SpdfTitle{Data: "My Article"}
    tpl.AddPage(210, 297)
    tpl.AddLayer("content", pdfgen.SpdfMargin{})
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "Article headline",
        Params: map[string]string{
            "font": "Helvetica",
            "size": "24",
            "x":    "10",
            "y":    "20",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "ftext",
        Data: "Long article body...",
        Params: map[string]string{
            "font": "Helvetica",
            "size": "12",
            "x":    "10",
            "y":    "40",
            "w":    "auto",
            "h":    "10",
        },
    })

    if err := tpl.RenderToFile("article.pdf"); err != nil {
        log.Fatal(err)
    }
}
```

### Invoice PDF

```go
package main

import (
    "log"

    "github.com/boomhut/pdf-gen"
)

func main() {
    tpl := pdfgen.NewTemplate("invoice")
    tpl.Title = pdfgen.SpdfTitle{Data: "Invoice"}
    tpl.AddPage(210, 297)
    tpl.AddLayer("content", pdfgen.SpdfMargin{})
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "Invoice #001",
        Params: map[string]string{
            "font": "Helvetica",
            "size": "18",
            "x":    "10",
            "y":    "20",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "Item A ........ $10",
        Params: map[string]string{
            "font": "Helvetica",
            "size": "12",
            "x":    "10",
            "y":    "40",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "Total: $10",
        Params: map[string]string{
            "font": "Helvetica-Bold",
            "size": "14",
            "x":    "10",
            "y":    "60",
        },
    })

    if err := tpl.RenderToFile("invoice.pdf"); err != nil {
        log.Fatal(err)
    }
}
```

### Event ticket PDF

```go
package main

import (
    "log"

    "github.com/boomhut/pdf-gen"
)

func main() {
    tpl := pdfgen.NewTemplate("ticket")
    tpl.AddPage(100, 50)
    tpl.AddLayer("content", pdfgen.SpdfMargin{})
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "Concert 2024",
        Params: map[string]string{
            "font": "Helvetica",
            "size": "14",
            "x":    "10",
            "y":    "10",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "qr",
        Data: "TICKET-ABC123",
        Params: map[string]string{
            "x":     "60",
            "y":     "10",
            "scale": "100",
        },
    })

    if err := tpl.RenderToFile("ticket.pdf"); err != nil {
        log.Fatal(err)
    }
}
```
