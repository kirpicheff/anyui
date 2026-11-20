package anyui

import (
	"testing"

	"github.com/kirpicheff/anyui/render"
	"github.com/kirpicheff/anyui/ui"
)

func TestBooleanOptionsUnification(t *testing.T) {
	// 1. Bold
	stBoldDef := ui.ApplyOptions([]ui.Option{Bold()})
	if stBoldDef.FontStyle != render.FontBold {
		t.Errorf("Bold(): expected FontBold, got %v", stBoldDef.FontStyle)
	}
	stBoldTrue := ui.ApplyOptions([]ui.Option{Bold(true)})
	if stBoldTrue.FontStyle != render.FontBold {
		t.Errorf("Bold(true): expected FontBold, got %v", stBoldTrue.FontStyle)
	}
	stBoldFalse := ui.ApplyOptions([]ui.Option{Bold(false)})
	if stBoldFalse.FontStyle != render.FontRegular {
		t.Errorf("Bold(false): expected FontRegular, got %v", stBoldFalse.FontStyle)
	}

	// 2. Primary
	stPriDef := ui.ApplyOptions([]ui.Option{Primary()})
	if stPriDef.Variant != ui.VariantPrimary {
		t.Errorf("Primary(): expected VariantPrimary, got %v", stPriDef.Variant)
	}
	stPriFalse := ui.ApplyOptions([]ui.Option{Primary(false)})
	if stPriFalse.Variant != ui.VariantDefault {
		t.Errorf("Primary(false): expected VariantDefault, got %v", stPriFalse.Variant)
	}

	// 3. Ghost
	stGhostDef := ui.ApplyOptions([]ui.Option{Ghost()})
	if stGhostDef.Variant != ui.VariantGhost {
		t.Errorf("Ghost(): expected VariantGhost, got %v", stGhostDef.Variant)
	}
	stGhostFalse := ui.ApplyOptions([]ui.Option{Ghost(false)})
	if stGhostFalse.Variant != ui.VariantDefault {
		t.Errorf("Ghost(false): expected VariantDefault, got %v", stGhostFalse.Variant)
	}

	// 4. Mono
	stMonoDef := ui.ApplyOptions([]ui.Option{Mono()})
	if stMonoDef.FontStyle != render.FontMono {
		t.Errorf("Mono(): expected FontMono, got %v", stMonoDef.FontStyle)
	}
	stMonoFalse := ui.ApplyOptions([]ui.Option{Mono(false)})
	if stMonoFalse.FontStyle != render.FontRegular {
		t.Errorf("Mono(false): expected FontRegular, got %v", stMonoFalse.FontStyle)
	}

	// 5. Disabled
	stDisDef := ui.ApplyOptions([]ui.Option{Disabled()})
	if !stDisDef.Disabled {
		t.Error("Disabled(): expected true")
	}
	stDisFalse := ui.ApplyOptions([]ui.Option{Disabled(false)})
	if stDisFalse.Disabled {
		t.Error("Disabled(false): expected false")
	}

	// 6. ReadOnly
	stRODef := ui.ApplyOptions([]ui.Option{ReadOnly()})
	if !stRODef.ReadOnly {
		t.Error("ReadOnly(): expected true")
	}
	stROFalse := ui.ApplyOptions([]ui.Option{ReadOnly(false)})
	if stROFalse.ReadOnly {
		t.Error("ReadOnly(false): expected false")
	}

	// 7. Clearable
	stClrDef := ui.ApplyOptions([]ui.Option{Clearable()})
	if !stClrDef.Clearable {
		t.Error("Clearable(): expected true")
	}
	stClrFalse := ui.ApplyOptions([]ui.Option{Clearable(false)})
	if stClrFalse.Clearable {
		t.Error("Clearable(false): expected false")
	}

	// 8. Indeterminate
	stIndDef := ui.ApplyOptions([]ui.Option{Indeterminate()})
	if !stIndDef.Indeterminate {
		t.Error("Indeterminate(): expected true")
	}
	stIndFalse := ui.ApplyOptions([]ui.Option{Indeterminate(false)})
	if stIndFalse.Indeterminate {
		t.Error("Indeterminate(false): expected false")
	}

	// 9. WordWrap
	stWWDef := ui.ApplyOptions([]ui.Option{WordWrap()})
	if !stWWDef.WordWrap {
		t.Error("WordWrap(): expected true")
	}
	stWWFalse := ui.ApplyOptions([]ui.Option{WordWrap(false)})
	if stWWFalse.WordWrap {
		t.Error("WordWrap(false): expected false")
	}

	// 10. ScrollToIndex & ScrollToID
	stScrollIdx := ui.ApplyOptions([]ui.Option{ScrollToIndex(42)})
	if stScrollIdx.ScrollToIndex == nil || *stScrollIdx.ScrollToIndex != 42 {
		t.Errorf("ScrollToIndex(42): expected 42, got %v", stScrollIdx.ScrollToIndex)
	}

	stScrollID := ui.ApplyOptions([]ui.Option{ScrollToID("my_anchor")})
	if stScrollID.ScrollToID != "my_anchor" {
		t.Errorf("ScrollToID: expected my_anchor, got %s", stScrollID.ScrollToID)
	}
}

