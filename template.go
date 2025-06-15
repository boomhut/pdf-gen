package pdfgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"os"
	"strconv"
	"strings"

	"codeberg.org/go-pdf/fpdf"
	"codeberg.org/go-pdf/fpdf/contrib/barcode"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

type SpdfTemplate struct {
	Name        string
	Pages       []SpdfPage
	Title       SpdfTitle
	Author      string
	Subject     SpdfSubject
	Keywords    SpdfKeywords
	DataBinding []SpdfDataKV
}

type SpdfTitle struct {
	Data   string
	Values []string
	Params map[string]string
}

type SpdfSubject struct {
	Data   string
	Values []string
	Params map[string]string
}

type SpdfKeywords struct {
	Data   string
	Values []string
	Params map[string]string
}

type SpdfPage struct {
	Width  int // in mm
	Height int // in mm
	Layers []SpdfLayer
}

type SpdfLayer struct {
	Name   string
	Items  []SpdfItem
	Margin SpdfMargin
}

type SpdfMargin struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

type SpdfItem struct {
	Type   string
	Data   string
	Values []string
	Params map[string]string
}

type SpdfDataKV struct {
	Key   string
	Value interface{}
	Get   func(key string) interface{}
}

type SpdfDataBinding struct {
	Set  func(key string, value interface{})
	Get  func(key string) interface{}
	data []SpdfDataKV
}

// RenderPdfTitle renders the title of the pdf template
func (tpl *SpdfTemplate) RenderPdfTitle() string {
	values := make([]interface{}, len(tpl.Title.Values))
	for i, v := range tpl.Title.Values {
		values[i] = v
	}
	return fmt.Sprintf(tpl.Title.Data, values...)
}

// RenderPdfSubject renders the subject of the pdf template
func (tpl *SpdfTemplate) RenderPdfSubject() string {
	values := make([]interface{}, len(tpl.Subject.Values))
	for i, v := range tpl.Subject.Values {
		values[i] = v
	}
	return fmt.Sprintf(tpl.Subject.Data, values...)
}

// RenderPdfKeywords renders the keywords of the pdf template
func (tpl *SpdfTemplate) RenderPdfKeywords() string {
	values := make([]interface{}, len(tpl.Keywords.Values))
	for i, v := range tpl.Keywords.Values {
		values[i] = v
	}
	return fmt.Sprintf(tpl.Keywords.Data, values...)
}

// RenderToFile renders the pdf template to a file
func (tpl *SpdfTemplate) RenderToFile(filename string) error {

	// create pdf
	pdf := fpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	// pdf.SetProtection(0, "", "abc")

	// set title and author
	pdf.SetTitle(tr(tpl.RenderPdfTitle()), true)
	pdf.SetSubject(tr(tpl.RenderPdfSubject()), true)
	pdf.SetKeywords(tr(tpl.RenderPdfKeywords()), true)

	pdf.SetAuthor(tr(tpl.Author), true)

	// creator and producer
	pdf.SetCreator("smartest.software pdf renderer", true)
	pdf.SetProducer("Logicos PDF", true)

	// load fonts
	pdf = MustReturnPtrFpdf(LoadFonts(pdf, "./resources/fonts", tpl))

	// add pages
	for _, page := range tpl.Pages {
		// pdf.AddPage()
		pdf.AddPageFormat("P", fpdf.SizeType{Wd: float64(page.Width), Ht: float64(page.Height)})
		for _, layer := range page.Layers {

			// Set margins for the layer (top, right, bottom, left)
			pdf.SetMargins(float64(layer.Margin.Left), float64(layer.Margin.Top), float64(layer.Margin.Right))

			// Initialize current Y position for auto calculation, take into account the top margin of the layer
			var currentY float64 = float64(layer.Margin.Top)
			for _, item := range layer.Items {
				if item.Type == "text" {
					// Get font size
					size, err := strconv.ParseFloat(item.Params["size"], 64)
					if err != nil {
						return err
					}
					pdf.SetFont(item.Params["font"], item.Params["style"], size)

					// Get text values
					values := make([]any, len(item.Values))
					for i, v := range item.Values {
						values[i] = v
					}
					text := fmt.Sprintf(item.Data, values...)
					trText := tr(text)

					// Get position (x, y)
					var x, y float64
					if item.Params["x"] != "" && item.Params["x"] != "auto" {
						x, err = strconv.ParseFloat(item.Params["x"], 64)
						if err != nil {
							return err
						}
					} else {
						x = pdf.GetX()
					}

					// if x = auto, then take left margin into account
					if item.Params["x"] == "auto" {
						x = float64(layer.Margin.Left)
					}

					// Get y position
					if item.Params["y"] != "" && item.Params["y"] != "auto" {
						y, err = strconv.ParseFloat(item.Params["y"], 64)
						if err != nil {
							return err
						}
						currentY = y
					} else {
						y = currentY
					}
					pdf.SetXY(x, y)

					// Get alignment
					align := item.Params["position"]
					if align == "" {
						align = "left"
					}

					// align is a string that can contain one or more of the following values:
					// left (default), right, center, baseline, top, middle, bottom or a combination of these values
					var alignStr string

					// check if align contains multiple values and set the alignment accordingly
					switch strings.Contains(align, " ") {
					case true:
						alignStr = ""
						alignArr := strings.Split(align, " ")
						for _, a := range alignArr {
							switch a {
							case "left":
								alignStr += "L"
							case "right":
								alignStr += "R"
							case "center":
								alignStr += "C"
							case "baseline":
								alignStr += "B"
							case "top":
								alignStr += "T"
							case "middle":
								alignStr += "M"
							case "bottom":
								alignStr += "B"
							}
						}
					case false:
						switch align {
						case "left":
							alignStr = "L"
						case "right":
							alignStr = "R"
						case "center":
							alignStr = "C"
						case "baseline":
							alignStr = "B"
						case "top":
							alignStr = "T"
						case "middle":
							alignStr = "M"
						case "bottom":
							alignStr = "B"
						}
					}

					// Get width and height
					var w, h float64
					if item.Params["w"] != "" && item.Params["w"] != "auto" {
						w, err = strconv.ParseFloat(item.Params["w"], 64)
						if err != nil {
							return err
						}
					} else if item.Params["w"] == "auto" {
						left, _, right, _ := pdf.GetMargins()
						pageWidth, _ := pdf.GetPageSize()
						w = pageWidth - left - right

					} else {
						w = pdf.GetStringWidth(trText)
					}
					if item.Params["h"] != "" {
						h, err = strconv.ParseFloat(item.Params["h"], 64)
						if err != nil {
							return err
						}
					} else {
						h = size / 2 // Default height
					}

					// Get border option
					border := item.Params["border"]

					// Set border color
					if item.Params["borderColor"] != "" {
						borderColor := item.Params["borderColor"]
						colors := make([]int, 3)
						for i, c := range strings.Split(borderColor, ",") {
							colorInt, err := strconv.Atoi(c)
							if err != nil {
								return err
							}
							colors[i] = colorInt
						}
						pdf.SetDrawColor(colors[0], colors[1], colors[2])
					}

					// Set text color
					if item.Params["color"] != "" {
						color := item.Params["color"]
						colors := make([]int, 3)
						for i, c := range strings.Split(color, ",") {
							colorInt, err := strconv.Atoi(c)
							if err != nil {
								return err
							}
							colors[i] = colorInt
						}
						pdf.SetTextColor(colors[0], colors[1], colors[2])
					}

					// Enable background fill if needed, default is false
					var enableBgFill bool
					if item.Params["bgFill"] == "true" {
						enableBgFill = true
					}

					// get fill color
					if item.Params["bgColor"] != "" {
						color := item.Params["bgColor"]
						colors := make([]int, 3)
						for i, c := range strings.Split(color, ",") {
							colorInt, err := strconv.Atoi(c)
							if err != nil {
								return err
							}
							colors[i] = colorInt
						}
						pdf.SetFillColor(colors[0], colors[1], colors[2])
					}

					// Get rotation
					var rotation float64
					if item.Params["rotate"] != "" {
						rotate, err := strconv.ParseFloat(item.Params["rotate"], 64)
						if err != nil {
							return err
						}
						rotation = rotate
					}

					// Apply rotation if needed
					if rotation != 0 {
						// Calculate text width and height for rotation pivot
						textWidth := pdf.GetStringWidth(trText)
						if item.Params["w"] != "auto" && w > 0 {
							textWidth = w
						}

						// Calculate center point for rotation
						centerX := x + (textWidth / 2)
						centerY := y + (h / 2)

						pdf.TransformBegin()
						pdf.TransformRotate(rotation, centerX, centerY)

						// Adjust position to maintain center alignment after rotation
						if alignStr == "C" {
							x = centerX - (textWidth / 2)
							pdf.SetX(x)
						}
					}

					// Output text with specified width, height, and border
					pdf.CellFormat(w, h, trText, border, 0, alignStr, enableBgFill, 0, "")

					// Reset rotation
					if rotation != 0 {
						pdf.TransformEnd()
					}

					// Update current Y position
					currentY += h * 1
				} else if item.Type == "ftext" {
					// use fpdf.Write() to write text with automatic line breaks
					// Get font size
					if item.Params["size"] == "" {
						item.Params["size"] = "12"
					}
					size, err := strconv.ParseFloat(item.Params["size"], 64)
					if err != nil {
						return err
					}
					pdf.SetFont(item.Params["font"], item.Params["style"], size)

					// Get text values
					values := make([]any, len(item.Values))
					for i, v := range item.Values {
						values[i] = v
					}
					text := fmt.Sprintf(item.Data, values...)
					trText := tr(text)

					// Get position (x, y)
					var x, y float64
					if item.Params["x"] != "" && item.Params["x"] != "auto" {
						x, err = strconv.ParseFloat(item.Params["x"], 64)
						if err != nil {
							return err
						}
					} else {
						x = pdf.GetX()
					}

					// if x = auto, then take left margin into account
					if item.Params["x"] == "auto" {
						x = float64(layer.Margin.Left)
					}

					// Get y position
					if item.Params["y"] != "" && item.Params["y"] != "auto" {
						y, err = strconv.ParseFloat(item.Params["y"], 64)
						if err != nil {
							return err
						}
						currentY = y
					} else {
						y = currentY
					}
					pdf.SetXY(x, y)

					// Get alignment
					align := item.Params["position"]
					if align == "" {
						align = "left"
					}

					// Get width and height
					var h float64

					if item.Params["h"] != "" {
						h, err = strconv.ParseFloat(item.Params["h"], 64)
						if err != nil {
							return err
						}
					}
					// text color
					if item.Params["color"] != "" {
						color := item.Params["color"]
						colors := make([]int, 3)
						for i, c := range strings.Split(color, ",") {
							colorInt, err := strconv.Atoi(c)
							if err != nil {
								return err
							}
							colors[i] = colorInt
						}
						pdf.SetTextColor(colors[0], colors[1], colors[2])
					}

					// write text with automatic line breaks
					pdf.Write(h, trText)

				} else if item.Type == "image" {

					// Get rotation
					var rotation float64
					// rotate, err := strconv.ParseFloat(item.Params["rotate"], 64)
					// if err != nil {
					// 	return err
					// }
					if item.Params["rotate"] != "" {
						rotate, err := strconv.ParseFloat(item.Params["rotate"], 64)
						if err != nil {
							return err
						}
						rotation = rotate
					} else {
						rotation = 0

					}

					// Get position
					x, err := strconv.ParseFloat(item.Params["x"], 64)
					if err != nil {
						return err
					}
					y, err := strconv.ParseFloat(item.Params["y"], 64)
					if err != nil {
						return err
					}

					var w, h float64
					if item.Params["w"] != "" && item.Params["h"] != "" {
						w, err = strconv.ParseFloat(item.Params["w"], 64)
						if err != nil {
							return err
						}
						h, err = strconv.ParseFloat(item.Params["h"], 64)
						if err != nil {
							return err
						}
					} else if item.Params["scale"] != "" {
						// Get original image dimensions
						info := pdf.RegisterImage(item.Data, "")
						if info == nil {
							return fmt.Errorf("failed to register image: %s", item.Data)
						}
						origW := info.Width()
						origH := info.Height()
						scale, err := strconv.ParseFloat(item.Params["scale"], 64)
						if err != nil {
							return err
						}
						// Calculate new dimensions
						w = origW * scale / 100
						h = origH * scale / 100
					} else {
						// Use original dimensions
						info := pdf.RegisterImage(item.Data, "")
						if info == nil {
							return fmt.Errorf("failed to register image: %s", item.Data)
						}
						w = info.Width()
						h = info.Height()
					}

					// Rotate image if needed
					if rotation != 0 {
						pdf.TransformBegin()
						pdf.TransformRotate(rotation, x, y)
					}

					// Add image to PDF
					pdf.ImageOptions(
						item.Data,
						x, y,
						w, h,
						false,
						fpdf.ImageOptions{},
						0,
						"",
					)

					// Reset rotation
					if rotation != 0 {
						pdf.TransformEnd()
					}
				} else if item.Type == "qr" {
					// first create qr code in temp file and then add it to the pdf
					// create temp file
					qrTempFile, err := os.CreateTemp("", "qr_*.png") // create temp RenderToFile
					if err != nil {
						return err
					}

					defer qrTempFile.Close()
					defer os.Remove(qrTempFile.Name())

					var itemColor color.RGBA

					if item.Params["color"] == "" {
						itemColor = color.RGBA{0, 0, 0, 255}
					} else {
						colors := make([]int, 3)
						for i, c := range strings.Split(item.Params["color"], ",") {
							colorInt, err := strconv.Atoi(c)
							if err != nil {
								return err
							}
							colors[i] = colorInt
						}
						itemColor = color.RGBA{uint8(colors[0]), uint8(colors[1]), uint8(colors[2]), 255}
					}

					// create qr code
					err = CreateQR(item.Data, item.Params["base"], qrTempFile.Name(), itemColor) // create qr code
					if err != nil {
						return err
					}

					// Now same as image
					// Get rotation
					var rotation float64
					// rotate, err := strconv.ParseFloat(item.Params["rotate"], 64)
					// if err != nil {
					// 	return err
					// }
					if item.Params["rotate"] != "" {
						rotate, err := strconv.ParseFloat(item.Params["rotate"], 64)
						if err != nil {
							return err
						}
						rotation = rotate
					} else {
						rotation = 0

					}

					// Get position
					x, err := strconv.ParseFloat(item.Params["x"], 64)
					if err != nil {
						return err
					}
					y, err := strconv.ParseFloat(item.Params["y"], 64)
					if err != nil {
						return err
					}

					var w, h float64
					if item.Params["w"] != "" && item.Params["h"] != "" {
						w, err = strconv.ParseFloat(item.Params["w"], 64)
						if err != nil {
							return err
						}
						h, err = strconv.ParseFloat(item.Params["h"], 64)
						if err != nil {
							return err
						}
					} else if item.Params["scale"] != "" {
						// Get original image dimensions
						info := pdf.RegisterImage(qrTempFile.Name(), "")
						if info == nil {
							return fmt.Errorf("failed to register image: %s", qrTempFile.Name())
						}
						origW := info.Width()
						origH := info.Height()
						scale, err := strconv.ParseFloat(item.Params["scale"], 64)
						if err != nil {
							return err
						}
						// Calculate new dimensions
						w = origW * scale / 100
						h = origH * scale / 100
					} else {
						// Use original dimensions
						info := pdf.RegisterImage(qrTempFile.Name(), "")
						if info == nil {
							return fmt.Errorf("failed to register image: %s", qrTempFile.Name())
						}
						w = info.Width()
						h = info.Height()
					}

					// Rotate image if needed
					if rotation != 0 {
						pdf.TransformBegin()
						pdf.TransformRotate(rotation, x, y)
					}

					// Add image to PDF
					pdf.ImageOptions(
						qrTempFile.Name(),
						x, y,
						w, h,
						false,
						fpdf.ImageOptions{},
						0,
						"",
					)

					// Reset rotation
					if rotation != 0 {
						pdf.TransformEnd()
					}

				} else if item.Type == "EAN13" || item.Type == "code128" || item.Type == "datamatrix" {
					// Barcode generation using contrib/barcode

					// Register barcode
					var key string
					switch item.Type {
					case "EAN13":
						key = barcode.RegisterEAN(pdf, item.Data)
					case "code128":
						key = barcode.RegisterCode128(pdf, item.Data)
					case "datamatrix":
						key = barcode.RegisterDataMatrix(pdf, item.Data)
					}

					// Get rotation
					var rotation float64
					if item.Params["rotate"] != "" {
						rotate, err := strconv.ParseFloat(item.Params["rotate"], 64)
						if err != nil {
							return err
						}
						rotation = rotate
					}

					// Get position
					x, err := strconv.ParseFloat(item.Params["x"], 64)
					if err != nil {
						return err
					}
					y, err := strconv.ParseFloat(item.Params["y"], 64)
					if err != nil {
						return err
					}

					var w, h float64
					if item.Params["w"] != "" && item.Params["h"] != "" {
						w, err = strconv.ParseFloat(item.Params["w"], 64)
						if err != nil {
							return err
						}
						h, err = strconv.ParseFloat(item.Params["h"], 64)
						if err != nil {
							return err
						}
					}

					if rotation != 0 {
						pdf.TransformBegin()
						pdf.TransformRotate(rotation, x, y)
					}

					barcode.Barcode(pdf, key, x, y, w, h, false)

					if rotation != 0 {
						pdf.TransformEnd()
					}

				}
			}
		}
	}

	// output pdf
	err := pdf.OutputFileAndClose(filename)
	if err != nil {
		return err
	}

	return nil
}

// CalcResize calculates the new width and height of an image
func CalcResize(width, height, maxWidth, maxHeight int) (int, int) {
	if width > maxWidth {
		height = height * maxWidth / width
		width = maxWidth
	}
	if height > maxHeight {
		width = width * maxHeight / height
		height = maxHeight
	}
	return width, height
}

// CalcResizeFromRatio calculates the new width and height of an image based on a ratio (e.g. 1:2) and percentage
func CalcResizeFromRatio(width, height int, ratio float64, percentage int) (int, int) {
	newWidth := int(float64(width) * (float64(percentage) / 100))
	newHeight := int(float64(height) * (float64(percentage) / 100) * ratio)
	return newWidth, newHeight
}

// CalcResizeFromWidth calculates the new width and height of an image based on a new width
func CalcResizeFromWidth(width, height, newWidth int) (int, int) {
	newHeight := height * newWidth / width
	return newWidth, newHeight
}

type fontResourceType struct {
}

func (f fontResourceType) Open(name string) (rdr io.Reader, err error) {
	// fmt.Printf("Generalized font loader opening %s\n", name)
	var buf []byte
	buf, err = os.ReadFile(name)
	if err == nil {
		rdr = bytes.NewReader(buf)
		// fmt.Printf("Generalized font loader reading %s\n", name)
	}
	return
}

// LoadFonts all fonts in the given directory
func LoadFonts(pdf *fpdf.Fpdf, dir string, tpl *SpdfTemplate) (*fpdf.Fpdf, error) {
	// // load fonts
	var fr fontResourceType
	pdf.SetFontLoader(fr)
	// pdf.AddFont("Calligrapher", "", "calligra.json")

	// get fonts.json in dir
	file, err := os.Open(dir + "/fonts.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// read fonts.json = []sFont
	type sFont struct {
		Name string
		Path string
	}

	sFonts := []sFont{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&sFonts)
	if err != nil {
		return nil, err
	}

	// get all fonts used in the template
	fonts := make(map[string]bool)
	for _, page := range tpl.Pages {
		for _, layer := range page.Layers {
			for _, item := range layer.Items {
				if item.Params["font"] != "" {
					fonts[item.Params["font"]] = true
				}
			}
		}
	}

	// add fonts if used in the template
	for font := range fonts {
		for _, f := range sFonts {
			if f.Name == font {
				pdf.AddFont(f.Name, "", dir+"/"+f.Path)
			}
		}
	}

	// // add fonts
	// for _, font := range sFonts {
	// 	pdf.AddFont(font.Name, "", dir+"/"+font.Path)
	// }

	return pdf, nil

}

// ListFonts lists all fonts in the given directory
func ListFonts(dir string) ([]string, error) {
	// get fonts.json in dir
	file, err := os.Open(dir + "/fonts.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// read fonts.json = []sFont
	type sFont struct {
		Name string
		Path string
	}

	sFonts := []sFont{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&sFonts)
	if err != nil {
		return nil, err
	}

	// print fonts for debugging
	fonts := []string{}
	for _, font := range sFonts {
		fonts = append(fonts, font.Name)
	}

	return fonts, nil
}

// MustReturnPtrFpdf returns a pointer to a fpdf instance, panics on error
func MustReturnPtrFpdf(iets any, err error) *fpdf.Fpdf {
	if err != nil {
		panic(err)
	}
	return iets.(*fpdf.Fpdf)
}

// NperPage splits a slice of items into n items per page
func NperPage(items []SpdfItem, n int) [][]SpdfItem {
	var pages [][]SpdfItem
	for i := 0; i < len(items); i += n {
		end := i + n
		if end > len(items) {
			end = len(items)
		}
		pages = append(pages, items[i:end])
	}
	return pages
}

// CreateQR creates a QR code
func CreateQR(data string, inputFilename, outputFilename string, fColor color.RGBA) error {
	qrc, err := qrcode.New(data)
	if err != nil {
		return err
	}

	options := []standard.ImageOption{
		// standard.WithHalftone(inputFilename),
		standard.WithQRWidth(15),
		// standard.WithHalftone(inputFilename),
		standard.WithBuiltinImageEncoder(standard.PNG_FORMAT),
		standard.WithBgTransparent(),
		standard.WithBgColor(color.RGBA{255, 255, 255, 255}),
		standard.WithFgColor(fColor),
	}

	w0, err := standard.New(outputFilename, options...)
	if err != nil {
		return err
	}

	err = qrc.Save(w0)
	if err != nil {
		return err
	}

	return nil
}

// NewTemplate creates a new pdf template
func NewTemplate(name string) *SpdfTemplate {
	return &SpdfTemplate{
		Name: name,
	}
}

// AddPage adds a page to the pdf template
func (tpl *SpdfTemplate) AddPage(width, height int) {
	tpl.Pages = append(tpl.Pages, SpdfPage{
		Width:  width,
		Height: height,
	})
}

// AddLayer adds a layer to the last page of the pdf template
func (tpl *SpdfTemplate) AddLayer(name string, margin SpdfMargin) {
	tpl.Pages[len(tpl.Pages)-1].Layers = append(tpl.Pages[len(tpl.Pages)-1].Layers, SpdfLayer{
		Name:   name,
		Margin: margin,
	})
}

// AddItem adds an item to the last layer of the last page of the pdf template
func (tpl *SpdfTemplate) AddItem(item SpdfItem) {
	tpl.Pages[len(tpl.Pages)-1].Layers[len(tpl.Pages[len(tpl.Pages)-1].Layers)-1].Items = append(tpl.Pages[len(tpl.Pages)-1].Layers[len(tpl.Pages[len(tpl.Pages)-1].Layers)-1].Items, item) // add item to last layer of last page

}

// AddDataBinding adds a data binding to the pdf template
func (tpl *SpdfTemplate) AddDataBinding(key string, value interface{}, get func(key string) interface{}) {
	tpl.DataBinding = append(tpl.DataBinding, SpdfDataKV{
		Key:   key,
		Value: value,
		Get:   get,
	})
}

// LoadTemplate loads a template from a json file and returns a pointer to the template
func LoadTemplate(filename string) (*SpdfTemplate, error) {
	// open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// read file
	decoder := json.NewDecoder(file)
	tpl := &SpdfTemplate{}
	err = decoder.Decode(tpl)
	if err != nil {
		return nil, err
	}

	return tpl, nil
}

// SaveTemplate saves a template to a json file
func SaveTemplate(tpl *SpdfTemplate, filename string) error {
	// create file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// write to file
	encoder := json.NewEncoder(file)
	err = encoder.Encode(tpl)
	if err != nil {
		return err
	}

	return nil
}

// LoadTemplateFromBytes loads a template from a byte slice and returns a pointer to the template
func LoadTemplateFromBytes(data []byte) (*SpdfTemplate, error) {
	tpl := &SpdfTemplate{}
	err := json.Unmarshal(data, tpl)
	if err != nil {
		return nil, err
	}

	return tpl, nil
}
