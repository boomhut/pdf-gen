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

### Advanced ticket PDF with barcode

```go
package main

import (
    "image/png"
    "log"
    "os"

    "github.com/boombuler/barcode"
    "github.com/boombuler/barcode/code128"
    "github.com/boomhut/pdf-gen"
)

func main() {
    // generate Code128 barcode image
    encoded, err := code128.Encode("EVENT-XYZ987654")
    if err != nil {
        log.Fatal(err)
    }
    code, err := barcode.Scale(encoded, 200, 60)
    if err != nil {
        log.Fatal(err)
    }
    f, err := os.Create("barcode.png")
    if err != nil {
        log.Fatal(err)
    }
    if err := png.Encode(f, code); err != nil {
        log.Fatal(err)
    }
    if err := f.Close(); err != nil {
        log.Fatal(err)
    }

    tpl := pdfgen.NewTemplate("adv-ticket")
    tpl.AddPage(200, 100)
    tpl.AddLayer("content", pdfgen.SpdfMargin{})
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "Summer Festival 2024",
        Params: map[string]string{
            "font":  "Helvetica",
            "style": "B",
            "size":  "20",
            "x":     "10",
            "y":     "15",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "Sector A, Row 5, Seat 20",
        Params: map[string]string{
            "font": "Helvetica",
            "size": "12",
            "x":    "10",
            "y":    "30",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "image",
        Data: "barcode.png",
        Params: map[string]string{
            "x":     "10",
            "y":     "50",
            "scale": "100",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "qr",
        Data: "ADV-TICKET-XYZ789",
        Params: map[string]string{
            "x":     "150",
            "y":     "40",
            "scale": "80",
        },
    })

    if err := tpl.RenderToFile("advanced_ticket.pdf"); err != nil {
        log.Fatal(err)
    }
}
```

### Ticket PDF with Code39 and EAN-13 barcodes

```go
package main

import (
    "image/png"
    "log"
    "os"

    "github.com/boombuler/barcode"
    "github.com/boombuler/barcode/code39"
    "github.com/boombuler/barcode/ean"
    "github.com/boomhut/pdf-gen"
)

func main() {
    // create Code39 barcode
    c39Data, err := code39.Encode("VIP12345", true, false)
    if err != nil {
        log.Fatal(err)
    }
    c39, err := barcode.Scale(c39Data, 180, 40)
    if err != nil {
        log.Fatal(err)
    }
    f1, err := os.Create("code39.png")
    if err != nil {
        log.Fatal(err)
    }
    if err := png.Encode(f1, c39); err != nil {
        log.Fatal(err)
    }
    if err := f1.Close(); err != nil {
        log.Fatal(err)
    }

    // create EAN-13 barcode
    eanData, err := ean.Encode("5901234123457")
    if err != nil {
        log.Fatal(err)
    }
    eanCode, err := barcode.Scale(eanData, 180, 60)
    if err != nil {
        log.Fatal(err)
    }
    f2, err := os.Create("ean13.png")
    if err != nil {
        log.Fatal(err)
    }
    if err := png.Encode(f2, eanCode); err != nil {
        log.Fatal(err)
    }
    if err := f2.Close(); err != nil {
        log.Fatal(err)
    }

    tpl := pdfgen.NewTemplate("vip-ticket")
    tpl.AddPage(210, 100)
    tpl.AddLayer("content", pdfgen.SpdfMargin{})
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "VIP Access",
        Params: map[string]string{
            "font":  "Helvetica",
            "style": "B",
            "size":  "22",
            "x":     "10",
            "y":     "15",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "text",
        Data: "Gate 3 - Seat 15",
        Params: map[string]string{
            "font": "Helvetica",
            "size": "12",
            "x":    "10",
            "y":    "30",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "image",
        Data: "code39.png",
        Params: map[string]string{
            "x":     "10",
            "y":     "50",
            "scale": "90",
        },
    })
    tpl.AddItem(pdfgen.SpdfItem{
        Type: "image",
        Data: "ean13.png",
        Params: map[string]string{
            "x":     "110",
            "y":     "50",
            "scale": "90",
        },
    })

    if err := tpl.RenderToFile("vip_ticket.pdf"); err != nil {
        log.Fatal(err)
    }
}
```
