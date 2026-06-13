//go:build darwin

// Hand-crafted additions to the trial appkit package.
// These expose singleton accessors and imperative methods that the
// code generator does not produce automatically.
package appkit

import (
	"unsafe"

	"github.com/ebitengine/purego/objc"

	raw "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/appkit"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/corefoundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
)

func ns(s string) *foundation.NSString {
	return foundation.NSStringStringWithUTF8String(s)
}

// ── Application extras ────────────────────────────────────────────────────────

// SharedApplication returns a trial wrapper around the NSApplication singleton.
func SharedApplication() *Application {
	return &Application{inner: raw.NSApplicationSharedApplication()}
}

// Activate brings the application to the foreground, ignoring other apps.
func (a *Application) Activate() {
	a.inner.ActivateIgnoringOtherApps(true)
}

// SetActivationPolicy sets the application activation policy.
func (a *Application) SetActivationPolicy(policy raw.NSApplicationActivationPolicy) {
	a.inner.SetActivationPolicy(policy)
}

// SetMainMenu sets the application's main menu bar.
func (a *Application) SetMainMenu(menu *Menu) {
	a.inner.SetMainMenu(menu.inner)
}

// SetWindowsMenu sets the Window menu used by NSApplication.
func (a *Application) SetWindowsMenu(menu *Menu) {
	a.inner.SetWindowsMenu(menu.inner)
}

// KeyWindow returns the current key window, or nil.
func (a *Application) KeyWindow() *Window {
	win := a.inner.KeyWindow()
	if win == nil {
		return nil
	}
	return &Window{inner: win}
}

// Run starts the NSApplication main run loop. Blocks until the app terminates.
func (a *Application) Run() { a.inner.Run() }

// ActivationPolicy returns the current activation policy.
func (a *Application) ActivationPolicy() raw.NSApplicationActivationPolicy {
	return a.inner.ActivationPolicy()
}

// Hide hides the application.
func (a *Application) Hide() { a.inner.Hide(0) }

// HideOtherApplications hides all other applications.
func (a *Application) HideOtherApplications() { a.inner.HideOtherApplications(0) }

// UnhideAllApplications shows all hidden applications.
func (a *Application) UnhideAllApplications() { a.inner.UnhideAllApplications(0) }

// Terminate requests that the application terminate.
func (a *Application) Terminate() { a.inner.Terminate(0) }

// ── Window extras ─────────────────────────────────────────────────────────────

// Show makes the window key and brings it to the front.
func (x *Window) Show() { x.inner.MakeKeyAndOrderFront(0) }

// OrderOut hides the window without releasing it.
func (x *Window) OrderOut() { x.inner.OrderOut(0) }

// Close closes the window.
func (x *Window) Close() { x.inner.Close() }

// Center centers the window on screen.
func (x *Window) Center() { x.inner.Center() }

// PerformClose simulates the user clicking the close button.
func (x *Window) PerformClose() { x.inner.PerformClose(0) }

// Miniaturize miniaturizes (minimises) the window.
func (x *Window) Miniaturize() { x.inner.Miniaturize(0) }

// ZoomWindow toggles the window between its standard and user-defined sizes.
func (x *Window) ZoomWindow() { x.inner.Zoom(0) }

// ToggleFullScreen toggles full-screen mode.
func (x *Window) ToggleFullScreen() { x.inner.ToggleFullScreen(0) }

// SetTitle sets the window title.
func (x *Window) SetTitle(title string) { x.inner.SetTitle(ns(title)) }

// SetMinSize sets the minimum window size.
func (x *Window) SetMinSize(width, height float64) {
	x.inner.SetMinSize(corefoundation.CGSize{Width: width, Height: height})
}

// SetFrameAutosaveName sets the autosave name used to persist the window frame.
func (x *Window) SetFrameAutosaveName(name string) {
	x.inner.SetFrameAutosaveName(ns(name))
}

// SetContentNSView sets the window's content view from a raw NSView pointer.
func (x *Window) SetContentNSView(nsv *raw.NSView) {
	x.inner.SetContentView(nsv)
}

var selBeginSheet = objc.RegisterName("beginSheet:completionHandler:")

// BeginSheet presents sheet as a sheet attached to the receiver.
//
// The raw BeginSheetCompletionHandler binding passes func(int) to objc.NewBlock,
// but purego requires the first parameter of a block callback to be the block's
// own pointer (func(unsafe.Pointer, int)). We bypass the raw binding here and
// call the ObjC selector directly with the correct block signature.
func (x *Window) BeginSheet(sheet *Window, completion func(int)) {
	if completion == nil {
		// Pass a nil block (0) directly — avoids NewBlock being called on a nil func.
		x.inner.Ptr().Send(selBeginSheet, sheet.inner.Ptr(), uintptr(0))
		return
	}
	// Wrap with the ObjC block calling convention: first arg is the block self ptr.
	handler := func(_ unsafe.Pointer, code int) { completion(code) }
	block := objc.NewBlock(handler)
	defer block.Release()
	x.inner.Ptr().Send(selBeginSheet, sheet.inner.Ptr(), block)
}

// EndSheet dismisses sheet from this window.
func (x *Window) EndSheet(sheet *Window) {
	x.inner.EndSheet(sheet.inner)
}

// Toolbar returns the window's toolbar as a trial wrapper, or nil.
func (x *Window) Toolbar() *Toolbar {
	tb := x.inner.Toolbar()
	if tb == nil {
		return nil
	}
	return &Toolbar{inner: tb}
}

// ── Toolbar extras ────────────────────────────────────────────────────────────

// IsVisible reports whether the toolbar is currently visible.
func (x *Toolbar) IsVisible() bool { return x.inner.IsVisible() }

// SetVisible shows or hides the toolbar.
func (x *Toolbar) SetVisible(visible bool) { x.inner.SetVisible(visible) }

// ── Image extras ──────────────────────────────────────────────────────────────

// ImageWithSystemSymbol returns an Image for the named SF Symbol.
func ImageWithSystemSymbol(name string) *Image {
	return &Image{inner: raw.NSImageImageWithSystemSymbolNameAccessibilityDescription(ns(name), ns(""))}
}

// SetTemplate marks or unmarks the image as a template image.
func (x *Image) SetTemplate(on bool) { x.inner.SetTemplate(on) }

// ── Menu extras ───────────────────────────────────────────────────────────────

// AddItem appends item to the menu.
func (x *Menu) AddItem(item *MenuItem) { x.inner.AddItem(item.inner) }

// AddSeparatorItem appends a separator line to the menu.
func (x *Menu) AddSeparatorItem() { x.inner.AddItem(raw.NSMenuItemSeparatorItem()) }

// SetSubmenuForItem attaches submenu to item and adds the relationship.
func (x *Menu) SetSubmenuForItem(submenu *Menu, item *MenuItem) {
	x.inner.SetSubmenuForItem(submenu.inner, item.inner)
}

// ── MenuItem extras ───────────────────────────────────────────────────────────

// NewMenuItemWithTitle creates a plain menu item with the given title.
func NewMenuItemWithTitle(title string) *MenuItem {
	id := objc.Send[objc.ID](objc.ID(objc.GetClass("NSMenuItem")), objc.RegisterName("new"))
	m := &MenuItem{inner: raw.NSMenuItemFromID(id)}
	m.inner.SetTitle(ns(title))
	return m
}

// SetEnabled enables or disables the menu item.
func (x *MenuItem) SetEnabled(on bool) { x.inner.SetEnabled(on) }

// SetTarget sets the action target.
func (x *MenuItem) SetTarget(target objc.ID) { x.inner.SetTarget(target) }

// SetAction sets the action selector.
func (x *MenuItem) SetAction(sel objc.SEL) { x.inner.SetAction(sel) }

// SetKeyEquivalent sets the key equivalent and modifier mask.
func (x *MenuItem) SetKeyEquivalent(key string, mods raw.NSEventModifierFlags) {
	x.inner.SetKeyEquivalent(ns(key))
	x.inner.SetKeyEquivalentModifierMask(mods)
}

// ── StatusBar extras ──────────────────────────────────────────────────────────

// SystemStatusBar returns the system-wide NSStatusBar.
func SystemStatusBar() *StatusBar {
	return &StatusBar{inner: raw.NSStatusBarSystemStatusBar()}
}

// StatusItemWithLength creates and returns a new status item with the given length.
// Pass -1 for NSVariableStatusItemLength.
func (x *StatusBar) StatusItemWithLength(length float64) *StatusItem {
	return &StatusItem{inner: x.inner.StatusItemWithLength(length)}
}

// RemoveStatusItem removes item from the status bar.
func (x *StatusBar) RemoveStatusItem(item *StatusItem) {
	x.inner.RemoveStatusItem(item.inner)
}

// ── StatusItem extras ─────────────────────────────────────────────────────────

// SetMenu attaches a pull-down menu to the status item.
func (x *StatusItem) SetMenu(menu *Menu) { x.inner.SetMenu(menu.inner) }

// SetImage sets the image displayed in the status bar button.
func (x *StatusItem) SetImage(img *Image) {
	if btn := x.inner.Button(); btn != nil {
		btn.SetImage(img.inner)
	}
}

// SetToolTip sets the tooltip on the status bar button.
func (x *StatusItem) SetToolTip(tip string) {
	if btn := x.inner.Button(); btn != nil {
		btn.SetToolTip(ns(tip))
	}
}

// Thickness returns the thickness of the status bar.
func (x *StatusBar) Thickness() float64 { return x.inner.Thickness() }
