package pdfgen

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"codeberg.org/go-pdf/fpdf"
)

func TestRenderPdfTitle(t *testing.T) {
	tpl := &SpdfTemplate{Title: SpdfTitle{Data: "Hello %s %s", Values: []string{"John", "Doe"}}}
	if got := tpl.RenderPdfTitle(); got != "Hello John Doe" {
		t.Errorf("expected 'Hello John Doe', got %s", got)
	}
}

func TestRenderPdfSubject(t *testing.T) {
	tpl := &SpdfTemplate{Subject: SpdfSubject{Data: "Subject %s", Values: []string{"One"}}}
	if got := tpl.RenderPdfSubject(); got != "Subject One" {
		t.Errorf("expected 'Subject One', got %s", got)
	}
}

func TestRenderPdfKeywords(t *testing.T) {
	tpl := &SpdfTemplate{Keywords: SpdfKeywords{Data: "K %s %s", Values: []string{"A", "B"}}}
	if got := tpl.RenderPdfKeywords(); got != "K A B" {
		t.Errorf("expected 'K A B', got %s", got)
	}
}

func TestAddPageLayerItem(t *testing.T) {
	tpl := NewTemplate("test")
	tpl.AddPage(100, 200)
	tpl.AddLayer("layer", SpdfMargin{Top: 1, Right: 2, Bottom: 3, Left: 4})
	tpl.AddItem(SpdfItem{Type: "text", Data: "hello"})

	if len(tpl.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(tpl.Pages))
	}
	page := tpl.Pages[0]
	if page.Width != 100 || page.Height != 200 {
		t.Errorf("unexpected page size: %+v", page)
	}
	if len(page.Layers) != 1 {
		t.Fatalf("expected 1 layer, got %d", len(page.Layers))
	}
	layer := page.Layers[0]
	if layer.Margin.Left != 4 {
		t.Errorf("unexpected margin: %+v", layer.Margin)
	}
	if len(layer.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(layer.Items))
	}
	if layer.Items[0].Data != "hello" {
		t.Errorf("unexpected item: %+v", layer.Items[0])
	}
}

func TestAddDataBinding(t *testing.T) {
	tpl := NewTemplate("d")
	tpl.AddDataBinding("k", "v", func(key string) interface{} { return "v" })
	if len(tpl.DataBinding) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(tpl.DataBinding))
	}
	if tpl.DataBinding[0].Key != "k" {
		t.Errorf("unexpected key: %s", tpl.DataBinding[0].Key)
	}
}

func TestNperPage(t *testing.T) {
	items := []SpdfItem{{}, {}, {}, {}, {}}
	pages := NperPage(items, 2)
	if len(pages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(pages))
	}
	if len(pages[0]) != 2 || len(pages[2]) != 1 {
		t.Errorf("unexpected split result: %+v", pages)
	}
}

func TestCalcResize(t *testing.T) {
	w, h := CalcResize(200, 100, 100, 80)
	if w != 100 || h != 50 {
		t.Errorf("unexpected size %d x %d", w, h)
	}
}

func TestCalcResizeFromRatio(t *testing.T) {
	w, h := CalcResizeFromRatio(100, 100, 2, 50)
	if w != 50 || h != 100 {
		t.Errorf("unexpected size %d x %d", w, h)
	}
}

func TestCalcResizeFromWidth(t *testing.T) {
	w, h := CalcResizeFromWidth(200, 100, 50)
	if w != 50 || h != 25 {
		t.Errorf("unexpected size %d x %d", w, h)
	}
}

func TestSaveAndLoadTemplate(t *testing.T) {
	tpl := NewTemplate("a")
	tmpdir := t.TempDir()
	file, err := os.CreateTemp(tmpdir, "tmpl-*.json")
	if err != nil {
		t.Fatalf("CreateTemp error: %v", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := SaveTemplate(tpl, file.Name()); err != nil {
		t.Fatalf("SaveTemplate error: %v", err)
	}
	loaded, err := LoadTemplate(file.Name())
	if err != nil {
		t.Fatalf("LoadTemplate error: %v", err)
	}
	if !reflect.DeepEqual(tpl, loaded) {
		t.Errorf("templates not equal: %+v vs %+v", tpl, loaded)
	}
}

func TestLoadTemplateFromBytes(t *testing.T) {
	tpl := NewTemplate("b")
	data, err := json.Marshal(tpl)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	loaded, err := LoadTemplateFromBytes(data)
	if err != nil {
		t.Fatalf("LoadTemplateFromBytes error: %v", err)
	}
	if !reflect.DeepEqual(tpl, loaded) {
		t.Errorf("templates not equal: %+v vs %+v", tpl, loaded)
	}
}

func TestListFonts(t *testing.T) {
	dir := t.TempDir()
	fontsJSON := `[{"Name":"A","Path":"a.json"},{"Name":"B","Path":"b.json"}]`
	if err := os.WriteFile(filepath.Join(dir, "fonts.json"), []byte(fontsJSON), 0o644); err != nil {
		t.Fatalf("write fonts.json: %v", err)
	}
	fonts, err := ListFonts(dir)
	if err != nil {
		t.Fatalf("ListFonts error: %v", err)
	}
	expected := []string{"A", "B"}
	if !reflect.DeepEqual(fonts, expected) {
		t.Errorf("expected %v, got %v", expected, fonts)
	}
}

func TestMustReturnPtrFpdf(t *testing.T) {
	pdf := &fpdf.Fpdf{}
	if MustReturnPtrFpdf(pdf, nil) != pdf {
		t.Errorf("did not return input pointer")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	MustReturnPtrFpdf(pdf, errors.New("fail"))
}

func TestRenderToFileWithFonts(t *testing.T) {
	// setup fonts directory expected by LoadFonts
	if err := os.MkdirAll("resources/fonts", 0o755); err != nil {
		t.Fatalf("mkdir fonts: %v", err)
	}
	if err := os.WriteFile("resources/fonts/fonts.json", []byte("[]"), 0o644); err != nil {
		t.Fatalf("write fonts.json: %v", err)
	}
	// Keep the resources directory so the generated PDF and fonts remain
	// available after tests for manual inspection.

	tpl := NewTemplate("font-test")
	tpl.AddPage(210, 297)
	tpl.AddLayer("layer", SpdfMargin{})

	tpl.AddItem(SpdfItem{Type: "text", Data: "Héllo ñ", Params: map[string]string{"font": "Courier", "size": "12", "x": "10", "y": "10"}})
	tpl.AddItem(SpdfItem{Type: "text", Data: "Ünicode", Params: map[string]string{"font": "Helvetica", "size": "14", "x": "10", "y": "20"}})
	tpl.AddItem(SpdfItem{Type: "text", Data: "Čau", Params: map[string]string{"font": "Times", "size": "16", "x": "10", "y": "30"}})

	// add barcode demo items
	tpl.AddItem(SpdfItem{Type: "EAN13", Data: "5901234123457", Params: map[string]string{"x": "10", "y": "50", "w": "40", "h": "20"}})
	tpl.AddItem(SpdfItem{Type: "code128", Data: "CODE128TEST", Params: map[string]string{"x": "60", "y": "50", "w": "40", "h": "20"}})
	tpl.AddItem(SpdfItem{Type: "datamatrix", Data: "DM1234", Params: map[string]string{"x": "110", "y": "50", "w": "20", "h": "20"}})

	// Write the output to a persistent file so it can be inspected after the
	// tests run.
	out := "demo.pdf"
	if err := tpl.RenderToFile(out); err != nil {
		t.Fatalf("RenderToFile error: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat pdf: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("pdf file is empty")
	}
	t.Logf("demo PDF generated: %s", out)
}

func TestFontResourceTypeOpen(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "data.txt")
	content := []byte("fontdata")
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	rdr, err := (fontResourceType{}).Open(file)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	got, err := io.ReadAll(rdr)
	if err != nil {
		t.Fatalf("read data: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("expected %q, got %q", content, got)
	}
	if _, err := (fontResourceType{}).Open(filepath.Join(dir, "missing")); err == nil {
		t.Errorf("expected error on missing file")
	}
}

func TestCreateQR(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "qr.png")
	if err := CreateQR("hello", "", out, color.RGBA{0, 0, 0, 255}); err != nil {
		t.Fatalf("CreateQR error: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat qr: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("qr output is empty")
	}
}

func TestLoadFontsMissingFontsJSON(t *testing.T) {
	dir := t.TempDir()
	pdf := fpdf.New("P", "mm", "A4", "")
	if _, err := LoadFonts(pdf, dir, &SpdfTemplate{}); err == nil {
		t.Errorf("expected error for missing fonts.json")
	}
}

func TestLoadTemplateFromBytesInvalid(t *testing.T) {
	if _, err := LoadTemplateFromBytes([]byte("{invalid")); err == nil {
		t.Errorf("expected error on invalid json")
	}
}

func TestListFontsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	// invalid JSON content
	if err := os.WriteFile(filepath.Join(dir, "fonts.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("write fonts.json: %v", err)
	}
	if _, err := ListFonts(dir); err == nil {
		t.Errorf("expected error for invalid json")
	}
}

func TestSaveTemplateError(t *testing.T) {
	tpl := NewTemplate("err")
	// path within non-existent directory
	fname := filepath.Join(t.TempDir(), "missing", "file.json")
	if err := SaveTemplate(tpl, fname); err == nil {
		t.Errorf("expected error when saving to invalid path")
	}
}

func TestLoadTemplateError(t *testing.T) {
	// missing file should return error
	if _, err := LoadTemplate(filepath.Join(t.TempDir(), "none.json")); err == nil {
		t.Errorf("expected error for missing file")
	}
}
