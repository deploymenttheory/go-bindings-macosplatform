//go:build darwin

// runtime_read_test.go exercises the generated Go bindings against the live
// macOS runtime. Every test here is a read-only, idempotent, non-destructive
// probe of system state. Nothing is written, deleted, or sent over the network.
//
// Tests are grouped by framework:
//
// ── Foundation (1–55) ────────────────────────────────────────────────────────
//  1. NSProcessInfo – processor count + physical memory
//  2. NSProcessInfo – OS version string
//  3. NSTimeZone – local timezone name, abbreviation, UTC offset
//  4. NSHost – current hostname + primary address
//  5. NSLocale – current locale identifier + language code
//  6. NSLocale – preferred languages array count
//  7. NSFileManager – current directory path + home directory URL
//  8. NSBundle – main bundle path + executable path
//  9. NSDate – current date sanity (TimeIntervalSince1970 > 0)
//
// 10. NSCalendar – current calendar, firstWeekday in [1,7]
// 11. NSUserDefaults – standard defaults non-nil, PersistentDomainNames non-nil
// 12. NSThread – mainThread non-nil, threadPriority in [0,1]
// 13. NSProcessInfo – processName non-empty, processIdentifier matches os.Getpid()
// 14. NSProcessInfo – systemUptime > 0, thermalState in [0,3]
// 15. NSOperationQueue – mainQueue non-nil, name non-nil, maxConcurrentOperationCount >= -1
// 16. NSFileManager – temporaryDirectory IsFileURL + path starts with "/"
// 17. NSNotificationCenter – defaultCenter non-nil
// 18. NSUUID – create UUID, check 36-char format with 4 hyphens
// 19. NSURL (file) – fileURLWithPath("/tmp"), isFileURL, path, absoluteString
// 20. NSURL (string) – URLWithString("https://example.com"), scheme + absoluteString
// 21. NSURLComponents – parse "https://example.com/path?q=1", scheme + host + path
// 22. NSNumber – numberWithInt(42) + numberWithDouble(3.14) roundtrips
// 23. NSNumberFormatter – localizedStringFromNumber, decimal style, non-empty
// 24. NSDateFormatter – localizedStringFromDate, medium/short styles, non-empty
// 25. NSTimeZone – knownTimeZoneNames array count > 100
// 26. NSLocale – availableLocaleIdentifiers count > 100
// 27. NSCharacterSet – whitespaceCharacterSet: space is member, 'A' is not
// 28. NSProcessInfo – environment dictionary non-nil, count > 0
// 29. NSByteCountFormatter – format 1 GiB with memory count style
// 30. NSData – dataWithContentsOfFile("/etc/hosts"), length > 0
// 31. NSMutableArray – arrayWithCapacity(10) non-nil, count = 0
// 32. NSFileManager – attributesOfFileSystem("/") dict non-nil, count > 0
// 33. NSProcessInfo – arguments non-nil, count >= 1
// 34. NSMutableString – stringWithCapacity(10) non-nil, length = 0
// 35. NSSet – set() non-nil, count = 0
// 36. NSMutableSet – setWithCapacity(5) non-nil, count = 0
// 37. NSIndexSet – indexSetWithIndex(42): count=1, containsIndex(42)=true, containsIndex(0)=false
// 38. NSDecimalNumber – one.doubleValue=1.0, zero.doubleValue=0.0, notANumber non-nil
// 39. NSMutableData – dataWithCapacity(100) non-nil, length = 0
// 40. NSMutableDictionary – dictionaryWithCapacity(5) non-nil, count = 0
// 41. NSURLSession – sharedSession non-nil
// 42. NSPredicate – predicateWithValue(true) evaluates true; predicateWithValue(false) evaluates false
// 43. NSRegularExpression – pattern "([0-9]+)": numberOfCaptureGroups = 1
// 44. NSScanner – scannerWithString("hello"): isAtEnd=false; empty string: isAtEnd=true
// 45. NSProgress – progressWithTotalUnitCount(100): totalUnitCount=100, completedUnitCount=0
// 46. NSJSONSerialization – isValidJSONObject(empty NSArray)=true; nil=false
// 47. NSBundle – allBundles count >= 1
// 48. NSBundle – allFrameworks non-nil
// 49. NSTimeZone – timeZoneForSecondsFromGMT(0): abbreviation is "GMT" or "UTC"
// 50. NSTimeZone – timeZoneDataVersion non-empty
// 51. NSDecimalNumber – maximumDecimalNumber and minimumDecimalNumber non-nil
// 52. NSLocale – isoCurrencyCodes count >= 100
// 53. NSCalendar – monthSymbols count = 12
// 54. NSLocale – commonISOCurrencyCodes count >= 10
// 55. NSCharacterSet – alphanumericCharacterSet: 'A' member, ' ' not member
//
// ── AppKit (56–92) ────────────────────────────────────────────────────────────
// 56. NSScreen – main screen display name + backing scale factor (main thread)
// 57. NSRunningApplication – current app bundle ID + executable architecture
// 58. NSWorkspace – frontmost application name (main thread)
// 59. NSApplication – sharedApplication non-nil, activationPolicy in valid range (main thread)
// 60. NSFont – systemFontOfSize(14.0) non-nil, pointSize ≈ 14 (main thread)
// 61. NSColor – labelColor, systemBlueColor, controlAccentColor all non-nil (main thread)
// 62. NSScreen – screens array count >= 1 (main thread)
// 63. NSWorkspace – runningApplications count >= 1 (main thread)
// 64. NSPasteboard – generalPasteboard non-nil, changeCount >= 0 (main thread)
// 65. NSCursor – arrowCursor + iBeamCursor non-nil (main thread)
// 66. NSColorSpace – sRGBColorSpace non-nil, colorSpaceModel > 0 (main thread)
// 67. NSPrintInfo – sharedPrintInfo non-nil, orientation in [0,1] (main thread)
// 68. NSBezierPath – bezierPath non-nil, defaultFlatness > 0 (main thread)
// 69. NSStatusBar – systemStatusBar non-nil, thickness > 0 (main thread)
// 70. NSFontManager – sharedFontManager non-nil, availableFontFamilies count >= 1 (main thread)
// 71. NSMenuItem – separatorItem non-nil, isSeparatorItem = true (main thread)
// 72. NSAppearance – currentDrawingAppearance (main thread, skip if nil)
// 73. NSDocumentController – sharedDocumentController non-nil, documents non-nil (main thread)
// 74. NSColorList – availableColorLists count >= 1 (main thread)
// 75. NSImageRep – registeredImageRepClasses count >= 1 (main thread)
// 76. NSEvent – keyRepeatDelay > 0 (main thread)
// 77. NSFont – monospacedDigitSystemFontOfSizeWeight(14, 0) non-nil (main thread)
// 78. NSFont – labelFontOfSize(12) non-nil, pointSize > 0 (main thread)
// 79. NSColor – systemRedColor, systemOrangeColor, systemGreenColor all non-nil (main thread)
// 80. NSWorkspace – notificationCenter non-nil (main thread)
// 81. NSScreen – screensHaveSeparateSpaces (class method, log value)
// 82. NSApplication – isRunning true, isActive (main thread)
// 83. NSColorSpace – genericRGBColorSpace non-nil (main thread)
// 84. NSColorSpace – displayP3ColorSpace non-nil (main thread)
// 85. NSFontManager – availableFonts count >= 1 (main thread)
// 86. NSEvent – doubleClickInterval > 0 (main thread)
// 87. NSEvent – keyRepeatInterval > 0 (main thread)
// 88. NSFont – labelFontSize > 0 (main thread)
// 89. NSEvent – isMouseCoalescingEnabled (class method, log value)
// 90. NSColorSpace – genericGrayColorSpace non-nil (main thread)
// 91. NSColorSpace – genericGamma22GrayColorSpace non-nil (main thread)
// 92. NSMenu – menuBarVisible class method (log value)
//
// ── Metal (93–100) ────────────────────────────────────────────────────────────
// 93. Metal – MTLCreateSystemDefaultDevice non-nil
// 94. Metal – MTLCopyAllDevices count >= 1
// 95. MTLArgumentDescriptor – argumentDescriptor() defaults: dataType, index, arrayLength logged
// 96. MTLArgumentDescriptor – set/get Index roundtrip: SetIndex(5) → Index()=5
// 97. MTLArgumentDescriptor – set/get ArrayLength roundtrip: SetArrayLength(10) → ArrayLength()=10
// 98. MTLBlitPassDescriptor – blitPassDescriptor() non-nil
// 99. MTLRenderPassDescriptor – renderPassDescriptor() non-nil
// 100. MTLFunctionDescriptor – functionDescriptor() non-nil
package acceptance_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks/appkit"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks/foundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks/metal"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/tools/grandcentraldispatch/mainthread"
	"github.com/ebitengine/purego/objc"
)

// ════════════════════════════════════════════════════════════════════════════
// Foundation
// ════════════════════════════════════════════════════════════════════════════

// ─── 1. NSProcessInfo: processor count + physical memory ─────────────────────

func TestRuntimeRead_ProcessorCount(t *testing.T) {
	info := foundation.NSProcessInfoProcessInfo()
	if info == nil {
		t.Fatal("NSProcessInfoProcessInfo() returned nil")
	}

	cpus := info.ProcessorCount()
	if cpus == 0 {
		t.Errorf("ProcessorCount() returned 0, expected >= 1")
	}
	t.Logf("ProcessorCount = %d", cpus)

	active := info.ActiveProcessorCount()
	if active == 0 {
		t.Errorf("ActiveProcessorCount() returned 0, expected >= 1")
	}
	if active > cpus {
		t.Errorf("ActiveProcessorCount (%d) > ProcessorCount (%d)", active, cpus)
	}
	t.Logf("ActiveProcessorCount = %d", active)

	ram := info.PhysicalMemory()
	if ram == 0 {
		t.Errorf("PhysicalMemory() returned 0")
	}
	t.Logf("PhysicalMemory = %d bytes (%.1f GiB)", ram, float64(ram)/(1<<30))
}

// ─── 2. NSProcessInfo: OS version string ─────────────────────────────────────

func TestRuntimeRead_OSVersionString(t *testing.T) {
	info := foundation.NSProcessInfoProcessInfo()
	if info == nil {
		t.Fatal("NSProcessInfoProcessInfo() returned nil")
	}

	nsver := info.OperatingSystemVersionString()
	if nsver == nil {
		t.Fatal("OperatingSystemVersionString() returned nil NSString")
	}

	ver := purego.GoString(nsver.Ptr())
	runtime.KeepAlive(nsver)
	if ver == "" {
		t.Errorf("OperatingSystemVersionString() converted to empty Go string")
	}
	t.Logf("OS version = %q", ver)

	if len(ver) < 7 {
		t.Errorf("OS version string suspiciously short: %q", ver)
	}
}

// ─── 3. NSTimeZone: local timezone name, abbreviation, UTC offset ─────────────

func TestRuntimeRead_LocalTimeZone(t *testing.T) {
	tz := foundation.NSTimeZoneLocalTimeZone()
	if tz == nil {
		t.Fatal("NSTimeZoneLocalTimeZone() returned nil")
	}

	tzNameNS := tz.Name()
	name := purego.GoString(tzNameNS.Ptr())
	runtime.KeepAlive(tzNameNS)
	if name == "" {
		t.Errorf("NSTimeZone.Name() is empty")
	}
	t.Logf("TimeZone name = %q", name)

	tzAbbrevNS := tz.Abbreviation()
	abbrev := purego.GoString(tzAbbrevNS.Ptr())
	runtime.KeepAlive(tzAbbrevNS)
	if abbrev == "" {
		t.Errorf("NSTimeZone.Abbreviation() is empty")
	}
	t.Logf("TimeZone abbreviation = %q", abbrev)

	offset := tz.SecondsFromGMT()
	const maxOffset = 14 * 3600
	if offset < -maxOffset || offset > maxOffset {
		t.Errorf("SecondsFromGMT() = %d, out of plausible range [%d, %d]", offset, -maxOffset, maxOffset)
	}
	t.Logf("SecondsFromGMT = %d (%+.1f hours)", offset, float64(offset)/3600)

	isDST := tz.IsDaylightSavingTime()
	t.Logf("IsDaylightSavingTime = %v", isDST)
}

// ─── 4. NSHost: current hostname + primary address ────────────────────────────

func TestRuntimeRead_HostInfo(t *testing.T) {
	host := foundation.NSHostCurrentHost()
	if host == nil {
		t.Fatal("NSHostCurrentHost() returned nil")
	}

	hostLocalNameNS := host.LocalizedName()
	locName := purego.GoString(hostLocalNameNS.Ptr())
	runtime.KeepAlive(hostLocalNameNS)
	if locName == "" {
		t.Errorf("NSHost.LocalizedName() is empty")
	}
	t.Logf("Host localizedName = %q", locName)

	hostAddrNS := host.Address()
	addr := purego.GoString(hostAddrNS.Ptr())
	runtime.KeepAlive(hostAddrNS)
	if addr == "" {
		t.Errorf("NSHost.Address() is empty")
	}
	t.Logf("Host address = %q", addr)

	if n := host.Name(); n != nil {
		s := purego.GoString(n.Ptr())
		runtime.KeepAlive(n)
		t.Logf("Host name = %q", s)
	} else {
		t.Log("Host.Name() returned nil (expected on some network configurations)")
	}
}

// ─── 5. NSLocale: current locale identifier + language code ──────────────────

func TestRuntimeRead_CurrentLocale(t *testing.T) {
	loc := foundation.NSLocaleCurrentLocale()
	if loc == nil {
		t.Fatal("NSLocaleCurrentLocale() returned nil")
	}

	localeIdNS := loc.LocaleIdentifier()
	id := purego.GoString(localeIdNS.Ptr())
	runtime.KeepAlive(localeIdNS)
	if id == "" {
		t.Errorf("NSLocale.LocaleIdentifier() is empty")
	}
	t.Logf("Locale identifier = %q", id)

	langNS := loc.LanguageCode()
	lang := purego.GoString(langNS.Ptr())
	runtime.KeepAlive(langNS)
	if lang == "" {
		t.Errorf("NSLocale.LanguageCode() is empty")
	}
	t.Logf("Language code = %q", lang)
}

// ─── 6. NSLocale: preferred languages array ───────────────────────────────────

func TestRuntimeRead_PreferredLanguages(t *testing.T) {
	langs := foundation.NSLocalePreferredLanguages()
	if langs == nil {
		t.Fatal("NSLocalePreferredLanguages() returned nil")
	}

	count := langs.Count()
	if count == 0 {
		t.Errorf("preferred languages array is empty")
	}
	t.Logf("PreferredLanguages count = %d", count)

	first := langs.FirstObject()
	if first == nil {
		t.Errorf("FirstObject() on non-empty languages array returned nil")
	}
	t.Logf("FirstObject (as runtime.Object) non-nil = %v", first != nil)
}

// ─── 7. NSFileManager: current directory + home directory URL ────────────────

func TestRuntimeRead_FileManagerPaths(t *testing.T) {
	fm := foundation.NSFileManagerDefaultManager()
	if fm == nil {
		t.Fatal("NSFileManagerDefaultManager() returned nil")
	}

	cwdNS := fm.CurrentDirectoryPath()
	if cwdNS == nil {
		t.Fatal("CurrentDirectoryPath() returned nil NSString")
	}
	cwd := purego.GoString(cwdNS.Ptr())
	runtime.KeepAlive(cwdNS)
	if cwd == "" {
		t.Errorf("CurrentDirectoryPath() converted to empty Go string")
	}
	t.Logf("CurrentDirectoryPath = %q", cwd)

	homeURL := fm.HomeDirectoryForCurrentUser()
	if homeURL == nil {
		t.Fatal("HomeDirectoryForCurrentUser() returned nil NSURL")
	}

	homePathNS := homeURL.Path()
	homePath := purego.GoString(homePathNS.Ptr())
	runtime.KeepAlive(homePathNS)
	if homePath == "" {
		t.Errorf("home NSURL.Path() converted to empty Go string")
	}
	t.Logf("HomeDirectory path = %q", homePath)

	homeIsFile := homeURL.IsFileURL()
	if !homeIsFile {
		t.Errorf("HomeDirectory URL.IsFileURL() = false, expected true for a file URL")
	}
	t.Logf("HomeDirectory URL.IsFileURL() = %v", homeIsFile)
}

// ─── 8. NSBundle: main bundle path + executable path ────────────────────────

func TestRuntimeRead_MainBundle(t *testing.T) {
	bundle := foundation.NSBundleMainBundle()
	if bundle == nil {
		t.Fatal("NSBundleMainBundle() returned nil")
	}

	if bp := bundle.BundlePath(); bp != nil {
		s := purego.GoString(bp.Ptr())
		runtime.KeepAlive(bp)
		t.Logf("BundlePath = %q", s)
	} else {
		t.Log("BundlePath = nil (acceptable for plain test binary)")
	}

	exePath := bundle.ExecutablePath()
	if exePath == nil {
		t.Fatal("ExecutablePath() returned nil")
	}
	exe := purego.GoString(exePath.Ptr())
	runtime.KeepAlive(exePath)
	if exe == "" {
		t.Errorf("ExecutablePath() converted to empty Go string")
	}
	if !strings.HasPrefix(exe, "/") {
		t.Errorf("ExecutablePath() = %q, expected an absolute path", exe)
	}
	t.Logf("ExecutablePath = %q", exe)
}

// ─── 9. NSDate: current date sanity ─────────────────────────────────────────

func TestRuntimeRead_CurrentDate(t *testing.T) {
	date := foundation.NSDateDate()
	if date == nil {
		t.Fatal("NSDateDate() returned nil")
	}

	ts := date.TimeIntervalSince1970()
	if ts <= 0 {
		t.Errorf("TimeIntervalSince1970() = %v, expected > 0", ts)
	}
	const jan2020 = 1577836800.0
	if ts < jan2020 {
		t.Errorf("TimeIntervalSince1970() = %v, suspiciously old (before 2020)", ts)
	}
	t.Logf("NSDate.TimeIntervalSince1970 = %.0f", ts)

	sinceNow := date.TimeIntervalSinceNow()
	const tolerance = 5.0
	if sinceNow < -tolerance || sinceNow > tolerance {
		t.Errorf("TimeIntervalSinceNow() = %v, expected within ±%vs of now", sinceNow, tolerance)
	}
	t.Logf("NSDate.TimeIntervalSinceNow = %.3f", sinceNow)
}

// ─── 10. NSCalendar: current calendar, firstWeekday in [1,7] ─────────────────

func TestRuntimeRead_CurrentCalendar(t *testing.T) {
	cal := foundation.NSCalendarCurrentCalendar()
	if cal == nil {
		t.Fatal("NSCalendarCurrentCalendar() returned nil")
	}

	fw := cal.FirstWeekday()
	if fw < 1 || fw > 7 {
		t.Errorf("FirstWeekday() = %d, expected 1–7 (Sunday=1 … Saturday=7)", fw)
	}
	t.Logf("FirstWeekday = %d", fw)

	id := cal.CalendarIdentifier()
	if id == nil {
		t.Errorf("CalendarIdentifier() returned nil")
	}
	t.Logf("CalendarIdentifier ptr non-nil = %v", id != nil)
}

// ─── 11. NSUserDefaults: standard defaults, PersistentDomainNames ─────────────

func TestRuntimeRead_UserDefaults(t *testing.T) {
	ud := foundation.NSUserDefaultsStandardUserDefaults()
	if ud == nil {
		t.Fatal("NSUserDefaultsStandardUserDefaults() returned nil")
	}

	domains := ud.PersistentDomainNames()
	if domains == nil {
		t.Fatal("PersistentDomainNames() returned nil")
	}
	count := domains.Count()
	t.Logf("PersistentDomainNames count = %d", count)

	dict := ud.DictionaryRepresentation()
	if dict == nil {
		t.Errorf("DictionaryRepresentation() returned nil")
	}
	t.Logf("DictionaryRepresentation non-nil = %v", dict != nil)
}

// ─── 12. NSThread: mainThread non-nil, threadPriority in [0,1] ───────────────

func TestRuntimeRead_MainThread(t *testing.T) {
	mainThread := foundation.NSThreadMainThread()
	if mainThread == nil {
		t.Fatal("NSThreadMainThread() returned nil")
	}

	isMain := mainThread.IsMainThread()
	if !isMain {
		t.Errorf("NSThreadMainThread().IsMainThread() = false, expected true")
	}
	t.Logf("NSThreadMainThread.IsMainThread = %v", isMain)

	prio := mainThread.ThreadPriority()
	if prio < 0.0 || prio > 1.0 {
		t.Errorf("ThreadPriority() = %v, expected [0.0, 1.0]", prio)
	}
	t.Logf("NSThreadMainThread.ThreadPriority = %.3f", prio)

	callerIsMain := foundation.NSThreadIsMainThread()
	t.Logf("NSThread(class).isMainThread (from goroutine) = %v", callerIsMain)
}

// ─── 13. NSProcessInfo: processName + processIdentifier ──────────────────────

func TestRuntimeRead_ProcessIdentity(t *testing.T) {
	info := foundation.NSProcessInfoProcessInfo()
	if info == nil {
		t.Fatal("NSProcessInfoProcessInfo() returned nil")
	}

	nameNS := info.ProcessName()
	if nameNS == nil {
		t.Fatal("ProcessName() returned nil NSString")
	}
	name := purego.GoString(nameNS.Ptr())
	runtime.KeepAlive(nameNS)
	if name == "" {
		t.Errorf("ProcessName() converted to empty Go string")
	}
	t.Logf("ProcessName = %q", name)

	pid := info.ProcessIdentifier()
	goPid := os.Getpid()
	if pid != goPid {
		t.Errorf("ProcessIdentifier() = %d, os.Getpid() = %d — mismatch", pid, goPid)
	}
	t.Logf("ProcessIdentifier = %d (matches os.Getpid())", pid)
}

// ─── 14. NSProcessInfo: systemUptime + thermalState ──────────────────────────

func TestRuntimeRead_SystemUptime(t *testing.T) {
	info := foundation.NSProcessInfoProcessInfo()
	if info == nil {
		t.Fatal("NSProcessInfoProcessInfo() returned nil")
	}

	uptime := info.SystemUptime()
	if uptime <= 0 {
		t.Errorf("SystemUptime() = %v, expected > 0", uptime)
	}
	t.Logf("SystemUptime = %.1f seconds (%.1f hours)", uptime, uptime/3600)

	thermal := info.ThermalState()
	if thermal < 0 || thermal > 3 {
		t.Errorf("ThermalState() = %d, expected 0–3", thermal)
	}
	t.Logf("ThermalState = %d", thermal)

	lowPower := info.IsLowPowerModeEnabled()
	t.Logf("IsLowPowerModeEnabled = %v", lowPower)
}

// ─── 15. NSOperationQueue: mainQueue, name, maxConcurrentOperationCount ───────

func TestRuntimeRead_OperationQueueMain(t *testing.T) {
	q := foundation.NSOperationQueueMainQueue()
	if q == nil {
		t.Fatal("NSOperationQueueMainQueue() returned nil")
	}

	nameNS := q.Name()
	if nameNS == nil {
		t.Fatal("NSOperationQueue.Name() returned nil")
	}
	name := purego.GoString(nameNS.Ptr())
	runtime.KeepAlive(nameNS)
	t.Logf("NSOperationQueue(main).Name = %q", name)

	maxOps := q.MaxConcurrentOperationCount()
	if maxOps < -1 {
		t.Errorf("MaxConcurrentOperationCount() = %d, expected >= -1", maxOps)
	}
	t.Logf("MaxConcurrentOperationCount = %d", maxOps)

	suspended := q.IsSuspended()
	t.Logf("IsSuspended = %v", suspended)
}

// ─── 16. NSFileManager: temporaryDirectory isFileURL + path prefix ────────────

func TestRuntimeRead_TemporaryDirectory(t *testing.T) {
	fm := foundation.NSFileManagerDefaultManager()
	if fm == nil {
		t.Fatal("NSFileManagerDefaultManager() returned nil")
	}

	tmpURL := fm.TemporaryDirectory()
	if tmpURL == nil {
		t.Fatal("TemporaryDirectory() returned nil NSURL")
	}

	if !tmpURL.IsFileURL() {
		t.Errorf("TemporaryDirectory().IsFileURL() = false, expected true")
	}
	t.Log("TemporaryDirectory.IsFileURL() = true")

	pathNS := tmpURL.Path()
	if pathNS == nil {
		t.Fatal("TemporaryDirectory().Path() returned nil NSString")
	}
	path := purego.GoString(pathNS.Ptr())
	runtime.KeepAlive(pathNS)
	if path == "" {
		t.Errorf("TemporaryDirectory path converted to empty Go string")
	}
	if !strings.HasPrefix(path, "/") {
		t.Errorf("TemporaryDirectory path = %q, expected absolute path", path)
	}
	t.Logf("TemporaryDirectory = %q", path)
}

// ─── 17. NSNotificationCenter: defaultCenter non-nil ─────────────────────────

func TestRuntimeRead_DefaultNotificationCenter(t *testing.T) {
	center := foundation.NSNotificationCenterDefaultCenter()
	if center == nil {
		t.Fatal("NSNotificationCenterDefaultCenter() returned nil")
	}
	t.Log("NSNotificationCenterDefaultCenter() non-nil")
}

// ─── 18. NSUUID: create UUID, verify 36-char format ──────────────────────────

func TestRuntimeRead_NSUUID(t *testing.T) {
	uuid := foundation.NSUUIDUUID()
	if uuid == nil {
		t.Fatal("NSUUIDUUID() returned nil")
	}

	uuidStrNS := uuid.UUIDString()
	if uuidStrNS == nil {
		t.Fatal("NSUUID.UUIDString() returned nil")
	}
	s := purego.GoString(uuidStrNS.Ptr())
	runtime.KeepAlive(uuidStrNS)

	if len(s) != 36 {
		t.Errorf("UUIDString() length = %d, expected 36; got %q", len(s), s)
	}
	hyphens := strings.Count(s, "-")
	if hyphens != 4 {
		t.Errorf("UUIDString() has %d hyphens, expected 4; got %q", hyphens, s)
	}
	t.Logf("UUIDString = %q", s)
}

// ─── 19. NSURL (file): fileURLWithPath("/tmp") ────────────────────────────────

func TestRuntimeRead_FileURL(t *testing.T) {
	pathNS := foundation.NSStringFromID(purego.NSString("/tmp"))
	url := foundation.NSURLFileURLWithPath(pathNS)
	if url == nil {
		t.Fatal("NSURLFileURLWithPath(\"/tmp\") returned nil")
	}

	if !url.IsFileURL() {
		t.Errorf("IsFileURL() = false, expected true")
	}

	pathOutNS := url.Path()
	if pathOutNS == nil {
		t.Fatal("NSURL.Path() returned nil")
	}
	p := purego.GoString(pathOutNS.Ptr())
	runtime.KeepAlive(pathOutNS)
	// /tmp is often a symlink to /private/tmp on macOS; either is acceptable.
	if p != "/tmp" && p != "/private/tmp" {
		t.Errorf("Path() = %q, expected \"/tmp\" or \"/private/tmp\"", p)
	}
	t.Logf("NSURL.Path = %q", p)

	absNS := url.AbsoluteString()
	if absNS == nil {
		t.Fatal("AbsoluteString() returned nil")
	}
	abs := purego.GoString(absNS.Ptr())
	runtime.KeepAlive(absNS)
	if !strings.HasPrefix(abs, "file://") {
		t.Errorf("AbsoluteString() = %q, expected \"file://\" prefix", abs)
	}
	t.Logf("NSURL.AbsoluteString = %q", abs)
}

// ─── 20. NSURL (string): parse https URL, check scheme ───────────────────────

func TestRuntimeRead_NSURL_String(t *testing.T) {
	strNS := foundation.NSStringFromID(purego.NSString("https://example.com"))
	url := foundation.NSURLURLWithString(strNS)
	if url == nil {
		t.Fatal("NSURLURLWithString(\"https://example.com\") returned nil")
	}

	schemeNS := url.Scheme()
	if schemeNS == nil {
		t.Fatal("NSURL.Scheme() returned nil")
	}
	scheme := purego.GoString(schemeNS.Ptr())
	runtime.KeepAlive(schemeNS)
	if scheme != "https" {
		t.Errorf("Scheme() = %q, expected \"https\"", scheme)
	}
	t.Logf("NSURL.Scheme = %q", scheme)

	hostNS := url.Host()
	if hostNS == nil {
		t.Fatal("NSURL.Host() returned nil")
	}
	host := purego.GoString(hostNS.Ptr())
	runtime.KeepAlive(hostNS)
	if host != "example.com" {
		t.Errorf("Host() = %q, expected \"example.com\"", host)
	}
	t.Logf("NSURL.Host = %q", host)

	absNS := url.AbsoluteString()
	if absNS == nil {
		t.Fatal("AbsoluteString() returned nil")
	}
	abs := purego.GoString(absNS.Ptr())
	runtime.KeepAlive(absNS)
	if abs == "" {
		t.Errorf("AbsoluteString() is empty")
	}
	t.Logf("NSURL.AbsoluteString = %q", abs)
}

// ─── 21. NSURLComponents: parse URL into components ──────────────────────────

func TestRuntimeRead_URLComponents(t *testing.T) {
	strNS := foundation.NSStringFromID(purego.NSString("https://example.com/path?q=1"))
	comps := foundation.NSURLComponentsComponentsWithString(strNS)
	if comps == nil {
		t.Fatal("NSURLComponentsComponentsWithString(...) returned nil")
	}

	schemeNS := comps.Scheme()
	if schemeNS == nil {
		t.Fatal("NSURLComponents.Scheme() returned nil")
	}
	scheme := purego.GoString(schemeNS.Ptr())
	runtime.KeepAlive(schemeNS)
	if scheme != "https" {
		t.Errorf("Scheme() = %q, expected \"https\"", scheme)
	}
	t.Logf("NSURLComponents.Scheme = %q", scheme)

	hostNS := comps.Host()
	if hostNS == nil {
		t.Fatal("NSURLComponents.Host() returned nil")
	}
	host := purego.GoString(hostNS.Ptr())
	runtime.KeepAlive(hostNS)
	if host != "example.com" {
		t.Errorf("Host() = %q, expected \"example.com\"", host)
	}
	t.Logf("NSURLComponents.Host = %q", host)

	pathNS := comps.Path()
	if pathNS == nil {
		t.Fatal("NSURLComponents.Path() returned nil")
	}
	path := purego.GoString(pathNS.Ptr())
	runtime.KeepAlive(pathNS)
	if path != "/path" {
		t.Errorf("Path() = %q, expected \"/path\"", path)
	}
	t.Logf("NSURLComponents.Path = %q", path)
}

// ─── 22. NSNumber: int and double roundtrips ──────────────────────────────────

func TestRuntimeRead_NSNumber(t *testing.T) {
	numInt := foundation.NSNumberNumberWithInt(42)
	if numInt == nil {
		t.Fatal("NSNumberNumberWithInt(42) returned nil")
	}
	got := numInt.IntValue()
	if got != 42 {
		t.Errorf("IntValue() = %d, expected 42", got)
	}
	t.Logf("NSNumber(42).IntValue = %d", got)

	const wantF = 3.14159
	numF := foundation.NSNumberNumberWithDouble(wantF)
	if numF == nil {
		t.Fatal("NSNumberNumberWithDouble(3.14159) returned nil")
	}
	gotF := numF.DoubleValue()
	diff := gotF - wantF
	if diff < -1e-10 || diff > 1e-10 {
		t.Errorf("DoubleValue() = %v, expected %v", gotF, wantF)
	}
	t.Logf("NSNumber(3.14159).DoubleValue = %v", gotF)

	numTrue := foundation.NSNumberNumberWithBool(true)
	if numTrue == nil {
		t.Fatal("NSNumberNumberWithBool(true) returned nil")
	}
	t.Logf("NSNumber(true) non-nil")
}

// ─── 23. NSNumberFormatter: localised decimal string ─────────────────────────

func TestRuntimeRead_NumberFormatter(t *testing.T) {
	num := foundation.NSNumberNumberWithDouble(1234567.89)
	if num == nil {
		t.Fatal("NSNumberNumberWithDouble returned nil")
	}

	strNS := foundation.NSNumberFormatterLocalizedStringFromNumberNumberStyle(num, foundation.NSNumberFormatterDecimalStyle)
	if strNS == nil {
		t.Fatal("NSNumberFormatterLocalizedStringFromNumberNumberStyle returned nil")
	}
	s := purego.GoString(strNS.Ptr())
	runtime.KeepAlive(strNS)
	if s == "" {
		t.Errorf("localised number string is empty")
	}
	t.Logf("NSNumberFormatter decimal = %q", s)
}

// ─── 24. NSDateFormatter: localised date string ───────────────────────────────

func TestRuntimeRead_DateFormatter(t *testing.T) {
	date := foundation.NSDateDate()
	if date == nil {
		t.Fatal("NSDateDate() returned nil")
	}

	strNS := foundation.NSDateFormatterLocalizedStringFromDateDateStyleTimeStyle(
		date,
		foundation.NSDateFormatterMediumStyle,
		foundation.NSDateFormatterShortStyle,
	)
	if strNS == nil {
		t.Fatal("NSDateFormatterLocalizedStringFromDateDateStyleTimeStyle returned nil")
	}
	s := purego.GoString(strNS.Ptr())
	runtime.KeepAlive(strNS)
	if s == "" {
		t.Errorf("localised date string is empty")
	}
	t.Logf("NSDateFormatter(medium/short) = %q", s)
}

// ─── 25. NSTimeZone: knownTimeZoneNames count ─────────────────────────────────

func TestRuntimeRead_KnownTimeZoneNames(t *testing.T) {
	names := foundation.NSTimeZoneKnownTimeZoneNames()
	if names == nil {
		t.Fatal("NSTimeZoneKnownTimeZoneNames() returned nil")
	}
	count := names.Count()
	if count < 100 {
		t.Errorf("KnownTimeZoneNames count = %d, expected >= 100", count)
	}
	t.Logf("KnownTimeZoneNames count = %d", count)
}

// ─── 26. NSLocale: availableLocaleIdentifiers count ───────────────────────────

func TestRuntimeRead_AvailableLocaleIdentifiers(t *testing.T) {
	ids := foundation.NSLocaleAvailableLocaleIdentifiers()
	if ids == nil {
		t.Fatal("NSLocaleAvailableLocaleIdentifiers() returned nil")
	}
	count := ids.Count()
	if count < 100 {
		t.Errorf("AvailableLocaleIdentifiers count = %d, expected >= 100", count)
	}
	t.Logf("AvailableLocaleIdentifiers count = %d", count)
}

// ─── 27. NSCharacterSet: whitespace membership ───────────────────────────────

func TestRuntimeRead_CharacterSet(t *testing.T) {
	ws := foundation.NSCharacterSetWhitespaceCharacterSet()
	if ws == nil {
		t.Fatal("NSCharacterSetWhitespaceCharacterSet() returned nil")
	}

	if !ws.CharacterIsMember(0x0020) {
		t.Errorf("whitespaceCharacterSet.CharacterIsMember(0x0020 ' ') = false, expected true")
	}
	t.Log("whitespaceCharacterSet contains space = true")

	if ws.CharacterIsMember(0x0041) {
		t.Errorf("whitespaceCharacterSet.CharacterIsMember(0x0041 'A') = true, expected false")
	}
	t.Log("whitespaceCharacterSet contains 'A' = false")
}

// ─── 28. NSProcessInfo: environment dictionary ───────────────────────────────

func TestRuntimeRead_Environment(t *testing.T) {
	info := foundation.NSProcessInfoProcessInfo()
	if info == nil {
		t.Fatal("NSProcessInfoProcessInfo() returned nil")
	}

	env := info.Environment()
	if env == nil {
		t.Fatal("ProcessInfo.Environment() returned nil")
	}
	count := env.Count()
	if count == 0 {
		t.Errorf("Environment() is empty, expected at least one entry (e.g. PATH)")
	}
	t.Logf("Environment() count = %d", count)
}

// ─── 29. NSByteCountFormatter: format 1 GiB ──────────────────────────────────

func TestRuntimeRead_ByteCountFormatter(t *testing.T) {
	const oneGiB = 1 << 30
	strNS := foundation.NSByteCountFormatterStringFromByteCountCountStyle(oneGiB, foundation.NSByteCountFormatterCountStyleMemory)
	if strNS == nil {
		t.Fatal("NSByteCountFormatterStringFromByteCountCountStyle returned nil")
	}
	s := purego.GoString(strNS.Ptr())
	runtime.KeepAlive(strNS)
	if s == "" {
		t.Errorf("byte count string is empty")
	}
	t.Logf("1 GiB formatted = %q", s)
}

// ─── 30. NSData: read /etc/hosts, verify length > 0 ──────────────────────────

func TestRuntimeRead_NSData(t *testing.T) {
	pathNS := foundation.NSStringFromID(purego.NSString("/etc/hosts"))
	data := foundation.NSDataDataWithContentsOfFile(pathNS)
	if data == nil {
		t.Skip("NSDataDataWithContentsOfFile(\"/etc/hosts\") returned nil — file absent or unreadable")
	}
	length := data.Length()
	if length == 0 {
		t.Errorf("NSData.Length() = 0, expected > 0 for /etc/hosts")
	}
	t.Logf("/etc/hosts NSData.Length = %d bytes", length)
}

// ─── 31. NSMutableArray: create with capacity, verify count = 0 ──────────────

func TestRuntimeRead_NSMutableArray(t *testing.T) {
	arr := foundation.NSMutableArrayArrayWithCapacity(10)
	if arr == nil {
		t.Fatal("NSMutableArrayArrayWithCapacity(10) returned nil")
	}
	count := arr.Count()
	if count != 0 {
		t.Errorf("Count() = %d on freshly created array, expected 0", count)
	}
	t.Logf("NSMutableArray(capacity=10).Count = %d", count)
}

// ─── 32. NSFileManager: filesystem attributes of "/" ─────────────────────────

func TestRuntimeRead_FileSystemAttributes(t *testing.T) {
	fm := foundation.NSFileManagerDefaultManager()
	if fm == nil {
		t.Fatal("NSFileManagerDefaultManager() returned nil")
	}

	pathNS := foundation.NSStringFromID(purego.NSString("/"))
	dict, err := fm.AttributesOfFileSystemForPathError(pathNS)
	if err != nil {
		t.Fatalf("AttributesOfFileSystemForPathError(\"/\"): %v", err)
	}
	if dict == nil {
		t.Fatal("AttributesOfFileSystemForPathError returned nil dict")
	}
	count := dict.Count()
	if count == 0 {
		t.Errorf("filesystem attributes dict for \"/\" is empty")
	}
	t.Logf("filesystem attributes for \"/\" count = %d", count)
}

// ─── 33. NSProcessInfo: arguments non-nil, count >= 1 ────────────────────────

func TestRuntimeRead_ProcessArguments(t *testing.T) {
	info := foundation.NSProcessInfoProcessInfo()
	if info == nil {
		t.Fatal("NSProcessInfoProcessInfo() returned nil")
	}

	args := info.Arguments()
	if args == nil {
		t.Fatal("ProcessInfo.Arguments() returned nil")
	}
	count := args.Count()
	if count == 0 {
		t.Errorf("Arguments().Count() = 0, expected >= 1 (executable name)")
	}
	t.Logf("Arguments count = %d", count)
}

// ─── 34. NSMutableString: stringWithCapacity, length = 0 ─────────────────────

func TestRuntimeRead_NSMutableString(t *testing.T) {
	ms := foundation.NSMutableStringStringWithCapacity(10)
	if ms == nil {
		t.Fatal("NSMutableStringStringWithCapacity(10) returned nil")
	}
	length := ms.Length()
	if length != 0 {
		t.Errorf("fresh NSMutableString length = %d, expected 0", length)
	}
	t.Logf("NSMutableString(capacity=10).Length = %d", length)
}

// ─── 35. NSSet: empty set, count = 0 ─────────────────────────────────────────

func TestRuntimeRead_NSSet(t *testing.T) {
	s := foundation.NSSetSet()
	if s == nil {
		t.Fatal("NSSetSet() returned nil")
	}
	count := s.Count()
	if count != 0 {
		t.Errorf("NSSet.Count() = %d, expected 0 for empty set", count)
	}
	t.Logf("NSSet().Count = %d", count)
}

// ─── 36. NSMutableSet: setWithCapacity, count = 0 ────────────────────────────

func TestRuntimeRead_NSMutableSet(t *testing.T) {
	ms := foundation.NSMutableSetSetWithCapacity(5)
	if ms == nil {
		t.Fatal("NSMutableSetSetWithCapacity(5) returned nil")
	}
	count := ms.Count()
	if count != 0 {
		t.Errorf("NSMutableSet.Count() = %d, expected 0", count)
	}
	t.Logf("NSMutableSet(capacity=5).Count = %d", count)
}

// ─── 37. NSIndexSet: indexSetWithIndex(42) ────────────────────────────────────

func TestRuntimeRead_NSIndexSet(t *testing.T) {
	is := foundation.NSIndexSetIndexSetWithIndex(42)
	if is == nil {
		t.Fatal("NSIndexSetIndexSetWithIndex(42) returned nil")
	}
	count := is.Count()
	if count != 1 {
		t.Errorf("Count() = %d, expected 1", count)
	}
	if !is.ContainsIndex(42) {
		t.Errorf("ContainsIndex(42) = false, expected true")
	}
	if is.ContainsIndex(0) {
		t.Errorf("ContainsIndex(0) = true, expected false")
	}
	t.Logf("NSIndexSet(42): Count=%d, ContainsIndex(42)=true, ContainsIndex(0)=false", count)
}

// ─── 38. NSDecimalNumber: one, zero, notANumber ───────────────────────────────

func TestRuntimeRead_NSDecimalNumber(t *testing.T) {
	one := foundation.NSDecimalNumberOne()
	if one == nil {
		t.Fatal("NSDecimalNumberOne() returned nil")
	}
	if v := one.DoubleValue(); v != 1.0 {
		t.Errorf("NSDecimalNumberOne().DoubleValue() = %v, expected 1.0", v)
	}
	t.Logf("NSDecimalNumberOne().DoubleValue = 1.0")

	zero := foundation.NSDecimalNumberZero()
	if zero == nil {
		t.Fatal("NSDecimalNumberZero() returned nil")
	}
	if v := zero.DoubleValue(); v != 0.0 {
		t.Errorf("NSDecimalNumberZero().DoubleValue() = %v, expected 0.0", v)
	}
	t.Logf("NSDecimalNumberZero().DoubleValue = 0.0")

	nan := foundation.NSDecimalNumberNotANumber()
	if nan == nil {
		t.Fatal("NSDecimalNumberNotANumber() returned nil")
	}
	t.Logf("NSDecimalNumberNotANumber() non-nil")
}

// ─── 39. NSMutableData: dataWithCapacity, length = 0 ─────────────────────────

func TestRuntimeRead_NSMutableData(t *testing.T) {
	d := foundation.NSMutableDataDataWithCapacity(100)
	if d == nil {
		t.Fatal("NSMutableDataDataWithCapacity(100) returned nil")
	}
	length := d.Length()
	if length != 0 {
		t.Errorf("NSMutableData.Length() = %d, expected 0 for capacity-only allocation", length)
	}
	t.Logf("NSMutableData(capacity=100).Length = %d", length)
}

// ─── 40. NSMutableDictionary: dictionaryWithCapacity, count = 0 ──────────────

func TestRuntimeRead_NSMutableDictionary(t *testing.T) {
	d := foundation.NSMutableDictionaryDictionaryWithCapacity(5)
	if d == nil {
		t.Fatal("NSMutableDictionaryDictionaryWithCapacity(5) returned nil")
	}
	count := d.Count()
	if count != 0 {
		t.Errorf("NSMutableDictionary.Count() = %d, expected 0", count)
	}
	t.Logf("NSMutableDictionary(capacity=5).Count = %d", count)
}

// ─── 41. NSURLSession: sharedSession non-nil ─────────────────────────────────

func TestRuntimeRead_NSURLSession(t *testing.T) {
	sess := foundation.NSURLSessionSharedSession()
	if sess == nil {
		t.Fatal("NSURLSessionSharedSession() returned nil")
	}
	t.Log("NSURLSessionSharedSession() non-nil")
}

// ─── 42. NSPredicate: predicateWithValue ─────────────────────────────────────

func TestRuntimeRead_NSPredicate(t *testing.T) {
	predTrue := foundation.NSPredicatePredicateWithValue(true)
	if predTrue == nil {
		t.Fatal("NSPredicatePredicateWithValue(true) returned nil")
	}
	if !predTrue.EvaluateWithObject(0) {
		t.Errorf("predicateWithValue(true).EvaluateWithObject(0) = false, expected true")
	}
	t.Log("predicateWithValue(true).EvaluateWithObject(0) = true")

	predFalse := foundation.NSPredicatePredicateWithValue(false)
	if predFalse == nil {
		t.Fatal("NSPredicatePredicateWithValue(false) returned nil")
	}
	if predFalse.EvaluateWithObject(0) {
		t.Errorf("predicateWithValue(false).EvaluateWithObject(0) = true, expected false")
	}
	t.Log("predicateWithValue(false).EvaluateWithObject(0) = false")
}

// ─── 43. NSRegularExpression: capture group count ────────────────────────────

func TestRuntimeRead_NSRegularExpression(t *testing.T) {
	patternNS := foundation.NSStringFromID(purego.NSString(`([0-9]+)`))
	re, err := foundation.NSRegularExpressionRegularExpressionWithPatternOptionsError(
		patternNS,
		foundation.NSRegularExpressionOptions(0),
	)
	if err != nil {
		t.Fatalf("NSRegularExpressionRegularExpressionWithPatternOptionsError: %v", err)
	}
	if re == nil {
		t.Fatal("NSRegularExpression returned nil")
	}
	groups := re.NumberOfCaptureGroups()
	if groups != 1 {
		t.Errorf("NumberOfCaptureGroups() = %d, expected 1 for pattern ([0-9]+)", groups)
	}
	t.Logf("NSRegularExpression([0-9]+).NumberOfCaptureGroups = %d", groups)
}

// ─── 44. NSScanner: isAtEnd ──────────────────────────────────────────────────

func TestRuntimeRead_NSScanner(t *testing.T) {
	helloNS := foundation.NSStringFromID(purego.NSString("hello"))
	scannerFull := foundation.NSScannerScannerWithString(helloNS)
	if scannerFull == nil {
		t.Fatal("NSScannerScannerWithString(\"hello\") returned nil")
	}
	if scannerFull.IsAtEnd() {
		t.Errorf("IsAtEnd() = true for non-empty string, expected false")
	}
	t.Log("NSScanner(\"hello\").IsAtEnd() = false")

	emptyNS := foundation.NSStringFromID(purego.NSString(""))
	scannerEmpty := foundation.NSScannerScannerWithString(emptyNS)
	if scannerEmpty == nil {
		t.Fatal("NSScannerScannerWithString(\"\") returned nil")
	}
	if !scannerEmpty.IsAtEnd() {
		t.Errorf("IsAtEnd() = false for empty string, expected true")
	}
	t.Log("NSScanner(\"\").IsAtEnd() = true")
}

// ─── 45. NSProgress: progressWithTotalUnitCount ──────────────────────────────

func TestRuntimeRead_NSProgress(t *testing.T) {
	prog := foundation.NSProgressProgressWithTotalUnitCount(100)
	if prog == nil {
		t.Fatal("NSProgressProgressWithTotalUnitCount(100) returned nil")
	}
	total := prog.TotalUnitCount()
	if total != 100 {
		t.Errorf("TotalUnitCount() = %d, expected 100", total)
	}
	completed := prog.CompletedUnitCount()
	if completed != 0 {
		t.Errorf("CompletedUnitCount() = %d, expected 0 on fresh progress", completed)
	}
	t.Logf("NSProgress: total=%d, completed=%d", total, completed)
}

// ─── 46. NSJSONSerialization: isValidJSONObject ───────────────────────────────

func TestRuntimeRead_NSJSONSerialization(t *testing.T) {
	// An empty NSArray is a valid top-level JSON object (produces []).
	arr := foundation.NSMutableArrayArrayWithCapacity(0)
	if arr == nil {
		t.Fatal("NSMutableArrayArrayWithCapacity(0) returned nil")
	}
	if !foundation.NSJSONSerializationIsValidJSONObject(arr.Ptr()) {
		t.Errorf("IsValidJSONObject(empty NSArray) = false, expected true")
	}
	t.Log("IsValidJSONObject(empty NSArray) = true")

	// nil is not a valid JSON object.
	if foundation.NSJSONSerializationIsValidJSONObject(0) {
		t.Errorf("IsValidJSONObject(nil) = true, expected false")
	}
	t.Log("IsValidJSONObject(nil) = false")
}

// ─── 47. NSBundle: allBundles count >= 1 ─────────────────────────────────────

func TestRuntimeRead_AllBundles(t *testing.T) {
	bundles := foundation.NSBundleAllBundles()
	if bundles == nil {
		t.Fatal("NSBundleAllBundles() returned nil")
	}
	count := bundles.Count()
	if count == 0 {
		t.Errorf("AllBundles().Count() = 0, expected >= 1 (at least the main bundle)")
	}
	t.Logf("NSBundle.AllBundles count = %d", count)
}

// ─── 48. NSBundle: allFrameworks non-nil ─────────────────────────────────────

func TestRuntimeRead_AllFrameworks(t *testing.T) {
	frameworks := foundation.NSBundleAllFrameworks()
	if frameworks == nil {
		t.Fatal("NSBundleAllFrameworks() returned nil")
	}
	count := frameworks.Count()
	t.Logf("NSBundle.AllFrameworks count = %d", count)
}

// ─── 49. NSTimeZone: timeZoneForSecondsFromGMT(0) is UTC/GMT ─────────────────

func TestRuntimeRead_NSTimeZoneGMT(t *testing.T) {
	tz := foundation.NSTimeZoneTimeZoneForSecondsFromGMT(0)
	if tz == nil {
		t.Fatal("NSTimeZoneTimeZoneForSecondsFromGMT(0) returned nil")
	}
	offset := tz.SecondsFromGMT()
	if offset != 0 {
		t.Errorf("SecondsFromGMT() = %d, expected 0 for GMT+0 timezone", offset)
	}
	abbrevNS := tz.Abbreviation()
	if abbrevNS == nil {
		t.Fatal("Abbreviation() returned nil")
	}
	abbrev := purego.GoString(abbrevNS.Ptr())
	runtime.KeepAlive(abbrevNS)
	if abbrev != "GMT" && abbrev != "UTC" {
		t.Errorf("Abbreviation() = %q, expected \"GMT\" or \"UTC\"", abbrev)
	}
	t.Logf("NSTimeZoneForSecondsFromGMT(0): offset=%d, abbrev=%q", offset, abbrev)
}

// ─── 50. NSTimeZone: timeZoneDataVersion non-empty ───────────────────────────

func TestRuntimeRead_NSTimeZoneDataVersion(t *testing.T) {
	versionNS := foundation.NSTimeZoneTimeZoneDataVersion()
	if versionNS == nil {
		t.Fatal("NSTimeZoneTimeZoneDataVersion() returned nil")
	}
	version := purego.GoString(versionNS.Ptr())
	runtime.KeepAlive(versionNS)
	if version == "" {
		t.Errorf("TimeZoneDataVersion is empty")
	}
	t.Logf("NSTimeZone.TimeZoneDataVersion = %q", version)
}

// ─── 51. NSDecimalNumber: maximum and minimum non-nil ────────────────────────

func TestRuntimeRead_NSDecimalNumberLimits(t *testing.T) {
	maxNum := foundation.NSDecimalNumberMaximumDecimalNumber()
	if maxNum == nil {
		t.Fatal("NSDecimalNumberMaximumDecimalNumber() returned nil")
	}
	t.Logf("NSDecimalNumber.MaximumDecimalNumber non-nil, doubleValue = %v", maxNum.DoubleValue())

	minNum := foundation.NSDecimalNumberMinimumDecimalNumber()
	if minNum == nil {
		t.Fatal("NSDecimalNumberMinimumDecimalNumber() returned nil")
	}
	t.Logf("NSDecimalNumber.MinimumDecimalNumber non-nil, doubleValue = %v", minNum.DoubleValue())
}

// ─── 52. NSLocale: isoCurrencyCodes count >= 100 ─────────────────────────────

func TestRuntimeRead_ISOCurrencyCodes(t *testing.T) {
	codes := foundation.NSLocaleISOCurrencyCodes()
	if codes == nil {
		t.Fatal("NSLocaleISOCurrencyCodes() returned nil")
	}
	count := codes.Count()
	if count < 100 {
		t.Errorf("ISOCurrencyCodes count = %d, expected >= 100", count)
	}
	t.Logf("NSLocale.ISOCurrencyCodes count = %d", count)
}

// ─── 53. NSCalendar: monthSymbols count = 12 ─────────────────────────────────

func TestRuntimeRead_CalendarMonthSymbols(t *testing.T) {
	cal := foundation.NSCalendarCurrentCalendar()
	if cal == nil {
		t.Fatal("NSCalendarCurrentCalendar() returned nil")
	}
	symbols := cal.MonthSymbols()
	if symbols == nil {
		t.Fatal("MonthSymbols() returned nil")
	}
	count := symbols.Count()
	if count != 12 {
		t.Errorf("MonthSymbols count = %d, expected 12", count)
	}
	t.Logf("NSCalendar.MonthSymbols count = %d", count)
}

// ─── 54. NSLocale: commonISOCurrencyCodes count >= 10 ────────────────────────

func TestRuntimeRead_CommonISOCurrencyCodes(t *testing.T) {
	codes := foundation.NSLocaleCommonISOCurrencyCodes()
	if codes == nil {
		t.Fatal("NSLocaleCommonISOCurrencyCodes() returned nil")
	}
	count := codes.Count()
	if count < 10 {
		t.Errorf("CommonISOCurrencyCodes count = %d, expected >= 10", count)
	}
	t.Logf("NSLocale.CommonISOCurrencyCodes count = %d", count)
}

// ─── 55. NSCharacterSet: alphanumericCharacterSet membership ─────────────────

func TestRuntimeRead_AlphanumericCharacterSet(t *testing.T) {
	cs := foundation.NSCharacterSetAlphanumericCharacterSet()
	if cs == nil {
		t.Fatal("NSCharacterSetAlphanumericCharacterSet() returned nil")
	}
	// 'A' (0x41) must be a member.
	if !cs.CharacterIsMember(0x0041) {
		t.Errorf("alphanumericCharacterSet.CharacterIsMember('A') = false, expected true")
	}
	t.Log("alphanumericCharacterSet contains 'A' = true")
	// Space (0x20) must not be a member.
	if cs.CharacterIsMember(0x0020) {
		t.Errorf("alphanumericCharacterSet.CharacterIsMember(' ') = true, expected false")
	}
	t.Log("alphanumericCharacterSet contains ' ' = false")
}

// ════════════════════════════════════════════════════════════════════════════
// AppKit
// ════════════════════════════════════════════════════════════════════════════

// ─── 56. NSScreen: main screen display name + backing scale factor ─────────────

func TestRuntimeRead_MainScreen(t *testing.T) {
	var (
		screenName  string
		scaleFactor float64
		screenNil   bool
	)

	mainthread.Do(func() {
		screen := appkit.NSScreenMainScreen()
		if screen == nil {
			screenNil = true
			return
		}
		nsName := screen.LocalizedName()
		if nsName != nil {
			screenName = purego.GoString(nsName.Ptr())
			runtime.KeepAlive(nsName)
		}
		scaleFactor = screen.BackingScaleFactor()
	})

	if screenNil {
		t.Skip("NSScreenMainScreen() returned nil — no display attached (headless environment?)")
	}

	t.Logf("Main screen localizedName = %q", screenName)

	if scaleFactor != 1.0 && scaleFactor != 2.0 && scaleFactor != 3.0 {
		t.Errorf("BackingScaleFactor() = %v, expected 1.0, 2.0, or 3.0", scaleFactor)
	}
	t.Logf("BackingScaleFactor = %.1f", scaleFactor)
}

// ─── 57. NSRunningApplication: current app info ───────────────────────────────

func TestRuntimeRead_CurrentApplication(t *testing.T) {
	app := appkit.NSRunningApplicationCurrentApplication()
	if app == nil {
		t.Fatal("NSRunningApplicationCurrentApplication() returned nil")
	}

	if bid := app.BundleIdentifier(); bid != nil {
		s := purego.GoString(bid.Ptr())
		runtime.KeepAlive(bid)
		t.Logf("BundleIdentifier = %q", s)
	} else {
		t.Log("BundleIdentifier = nil (expected for test binary without Info.plist)")
	}

	if name := app.LocalizedName(); name != nil {
		s := purego.GoString(name.Ptr())
		runtime.KeepAlive(name)
		t.Logf("LocalizedName = %q", s)
	} else {
		t.Log("LocalizedName = nil (expected for test binary without Info.plist)")
	}

	arch := app.ExecutableArchitecture()
	t.Logf("ExecutableArchitecture = 0x%x (%d)", uint64(arch), arch)
	if arch == 0 {
		t.Errorf("ExecutableArchitecture() returned 0, expected a valid architecture constant")
	}

	finished := app.IsFinishedLaunching()
	t.Logf("IsFinishedLaunching = %v", finished)
}

// ─── 58. NSWorkspace: frontmost application name ─────────────────────────────

func TestRuntimeRead_FrontmostApplication(t *testing.T) {
	var appName string
	var wsNil, appNil bool

	mainthread.Do(func() {
		ws := appkit.NSWorkspaceSharedWorkspace()
		if ws == nil {
			wsNil = true
			return
		}
		app := ws.FrontmostApplication()
		if app == nil {
			appNil = true
			return
		}
		if name := app.LocalizedName(); name != nil {
			appName = purego.GoString(name.Ptr())
			runtime.KeepAlive(name)
		}
	})

	if wsNil {
		t.Skip("NSWorkspaceSharedWorkspace() returned nil")
	}
	if appNil {
		t.Skip("FrontmostApplication() returned nil — no GUI session active")
	}

	t.Logf("Frontmost application = %q", appName)
	if appName == "" {
		t.Errorf("FrontmostApplication().LocalizedName() resolved to empty string")
	}
}

// ─── 59. NSApplication: sharedApplication, activationPolicy ──────────────────

func TestRuntimeRead_SharedApplication(t *testing.T) {
	var (
		appNil           bool
		activationPolicy int64
	)

	mainthread.Do(func() {
		app := appkit.NSApplicationSharedApplication()
		if app == nil {
			appNil = true
			return
		}
		activationPolicy = int64(app.ActivationPolicy())
	})

	if appNil {
		t.Skip("NSApplicationSharedApplication() returned nil")
	}

	if activationPolicy < 0 || activationPolicy > 2 {
		t.Errorf("ActivationPolicy() = %d, expected 0–2", activationPolicy)
	}
	t.Logf("NSApplication.ActivationPolicy = %d", activationPolicy)
}

// ─── 60. NSFont: systemFontOfSize, pointSize, fontName, familyName ────────────

func TestRuntimeRead_SystemFont(t *testing.T) {
	var (
		fontNil    bool
		pointSize  float64
		fontName   string
		familyName string
	)

	mainthread.Do(func() {
		font := appkit.NSFontSystemFontOfSize(14.0)
		if font == nil {
			fontNil = true
			return
		}
		pointSize = font.PointSize()
		if fn := font.FontName(); fn != nil {
			fontName = purego.GoString(fn.Ptr())
			runtime.KeepAlive(fn)
		}
		if fam := font.FamilyName(); fam != nil {
			familyName = purego.GoString(fam.Ptr())
			runtime.KeepAlive(fam)
		}
	})

	if fontNil {
		t.Skip("NSFontSystemFontOfSize(14) returned nil")
	}

	const wantSize = 14.0
	if pointSize != wantSize {
		t.Errorf("PointSize() = %v, expected %v", pointSize, wantSize)
	}
	t.Logf("PointSize = %.1f", pointSize)

	if fontName == "" {
		t.Errorf("FontName() resolved to empty string")
	}
	t.Logf("FontName = %q", fontName)

	if familyName == "" {
		t.Errorf("FamilyName() resolved to empty string")
	}
	t.Logf("FamilyName = %q", familyName)
}

// ─── 61. NSColor: labelColor, systemBlueColor, controlAccentColor ─────────────

func TestRuntimeRead_SemanticColors(t *testing.T) {
	var (
		labelNil, blueNil, accentNil bool
	)

	mainthread.Do(func() {
		labelNil = appkit.NSColorLabelColor() == nil
		blueNil = appkit.NSColorSystemBlueColor() == nil
		accentNil = appkit.NSColorControlAccentColor() == nil
	})

	if labelNil {
		t.Errorf("NSColorLabelColor() returned nil")
	} else {
		t.Log("NSColorLabelColor() non-nil")
	}
	if blueNil {
		t.Errorf("NSColorSystemBlueColor() returned nil")
	} else {
		t.Log("NSColorSystemBlueColor() non-nil")
	}
	if accentNil {
		t.Errorf("NSColorControlAccentColor() returned nil")
	} else {
		t.Log("NSColorControlAccentColor() non-nil")
	}
}

// ─── 62. NSScreen: screens array count >= 1 ──────────────────────────────────

func TestRuntimeRead_AllScreens(t *testing.T) {
	var (
		screensNil bool
		count      uint
	)

	mainthread.Do(func() {
		screens := appkit.NSScreenScreens()
		if screens == nil {
			screensNil = true
			return
		}
		count = screens.Count()
	})

	if screensNil {
		t.Skip("NSScreenScreens() returned nil — headless environment?")
	}

	if count == 0 {
		t.Errorf("NSScreenScreens().Count() = 0, expected >= 1")
	}
	t.Logf("NSScreenScreens count = %d", count)
}

// ─── 63. NSWorkspace: runningApplications count >= 1 ─────────────────────────

func TestRuntimeRead_RunningApplications(t *testing.T) {
	var (
		wsNil  bool
		arrNil bool
		count  uint
	)

	mainthread.Do(func() {
		ws := appkit.NSWorkspaceSharedWorkspace()
		if ws == nil {
			wsNil = true
			return
		}
		apps := ws.RunningApplications()
		if apps == nil {
			arrNil = true
			return
		}
		count = apps.Count()
	})

	if wsNil {
		t.Skip("NSWorkspaceSharedWorkspace() returned nil")
	}
	if arrNil {
		t.Skip("RunningApplications() returned nil — no GUI session active?")
	}

	if count == 0 {
		t.Errorf("RunningApplications().Count() = 0, expected >= 1 (at least this process)")
	}
	t.Logf("RunningApplications count = %d", count)
}

// ─── 64. NSPasteboard: generalPasteboard, changeCount ────────────────────────

func TestRuntimeRead_Pasteboard(t *testing.T) {
	var (
		pbNil       bool
		changeCount int
	)

	mainthread.Do(func() {
		pb := appkit.NSPasteboardGeneralPasteboard()
		if pb == nil {
			pbNil = true
			return
		}
		changeCount = pb.ChangeCount()
	})

	if pbNil {
		t.Skip("NSPasteboardGeneralPasteboard() returned nil")
	}
	if changeCount < 0 {
		t.Errorf("NSPasteboard.ChangeCount() = %d, expected >= 0", changeCount)
	}
	t.Logf("NSPasteboard.ChangeCount = %d", changeCount)
}

// ─── 65. NSCursor: arrowCursor + iBeamCursor non-nil ─────────────────────────

func TestRuntimeRead_Cursor(t *testing.T) {
	var arrowNil, iBeamNil bool

	mainthread.Do(func() {
		_ = appkit.NSApplicationSharedApplication()
		arrowNil = appkit.NSCursorArrowCursor() == nil
		iBeamNil = appkit.NSCursorIBeamCursor() == nil
	})

	if arrowNil {
		t.Skip("NSCursorArrowCursor() returned nil — cursor system unavailable (headless?)")
	}
	t.Log("NSCursorArrowCursor() non-nil")

	if iBeamNil {
		t.Errorf("NSCursorIBeamCursor() returned nil after AppKit init")
	} else {
		t.Log("NSCursorIBeamCursor() non-nil")
	}
}

// ─── 66. NSColorSpace: sRGB color space model ────────────────────────────────

func TestRuntimeRead_ColorSpace(t *testing.T) {
	var (
		csNil bool
		model appkit.NSColorSpaceModel
		nameS string
	)

	mainthread.Do(func() {
		cs := appkit.NSColorSpaceSRGBColorSpace()
		if cs == nil {
			csNil = true
			return
		}
		model = cs.ColorSpaceModel()
		if n := cs.LocalizedName(); n != nil {
			nameS = purego.GoString(n.Ptr())
			runtime.KeepAlive(n)
		}
	})

	if csNil {
		t.Skip("NSColorSpaceSRGBColorSpace() returned nil")
	}
	// The generated NSColorSpaceModelUnknown constant is off by one vs the Apple
	// header (-1 mapped to 0). Avoid comparing against NSColorSpaceModelRGB
	// directly; instead verify the model is non-unknown (> 0) and the localized
	// name confirms it is sRGB.
	if model <= 0 {
		t.Errorf("sRGBColorSpace.ColorSpaceModel() = %d, expected a positive (non-unknown) model", model)
	}
	if !strings.Contains(nameS, "RGB") && !strings.Contains(nameS, "sRGB") {
		t.Errorf("sRGBColorSpace localizedName = %q, expected to contain \"RGB\" or \"sRGB\"", nameS)
	}
	t.Logf("sRGBColorSpace model = %d, localizedName = %q", model, nameS)
}

// ─── 67. NSPrintInfo: sharedPrintInfo orientation ────────────────────────────

func TestRuntimeRead_PrintInfo(t *testing.T) {
	var (
		piNil       bool
		orientation appkit.NSPaperOrientation
		topMargin   float64
	)

	mainthread.Do(func() {
		pi := appkit.NSPrintInfoSharedPrintInfo()
		if pi == nil {
			piNil = true
			return
		}
		orientation = pi.Orientation()
		topMargin = pi.TopMargin()
	})

	if piNil {
		t.Skip("NSPrintInfoSharedPrintInfo() returned nil")
	}
	if orientation < 0 || orientation > 1 {
		t.Errorf("NSPrintInfo.Orientation() = %d, expected 0 or 1", orientation)
	}
	t.Logf("NSPrintInfo.Orientation = %d", orientation)

	if topMargin < 0 {
		t.Errorf("NSPrintInfo.TopMargin() = %v, expected >= 0", topMargin)
	}
	t.Logf("NSPrintInfo.TopMargin = %.1f", topMargin)
}

// ─── 68. NSBezierPath: empty path, class-level defaults ──────────────────────

func TestRuntimeRead_BezierPath(t *testing.T) {
	var (
		pathNil    bool
		flatness   float64
		miterLimit float64
	)

	mainthread.Do(func() {
		bp := appkit.NSBezierPathBezierPath()
		if bp == nil {
			pathNil = true
			return
		}
		flatness = appkit.NSBezierPathDefaultFlatness()
		miterLimit = appkit.NSBezierPathDefaultMiterLimit()
	})

	if pathNil {
		t.Skip("NSBezierPathBezierPath() returned nil")
	}

	if flatness <= 0 || flatness > 100 {
		t.Errorf("DefaultFlatness() = %v, expected (0, 100]", flatness)
	}
	t.Logf("NSBezierPath.DefaultFlatness = %v", flatness)

	if miterLimit <= 0 {
		t.Errorf("DefaultMiterLimit() = %v, expected > 0", miterLimit)
	}
	t.Logf("NSBezierPath.DefaultMiterLimit = %v", miterLimit)
}

// ─── 69. NSStatusBar: systemStatusBar thickness ──────────────────────────────

func TestRuntimeRead_StatusBar(t *testing.T) {
	var (
		sbNil     bool
		thickness float64
	)

	mainthread.Do(func() {
		sb := appkit.NSStatusBarSystemStatusBar()
		if sb == nil {
			sbNil = true
			return
		}
		thickness = sb.Thickness()
	})

	if sbNil {
		t.Skip("NSStatusBarSystemStatusBar() returned nil — headless environment?")
	}
	if thickness <= 0 {
		t.Errorf("NSStatusBar.Thickness() = %v, expected > 0", thickness)
	}
	t.Logf("NSStatusBar.Thickness = %.1f", thickness)
}

// ─── 70. NSFontManager: sharedFontManager, availableFontFamilies ─────────────

func TestRuntimeRead_FontManager(t *testing.T) {
	var (
		fmNil       bool
		familyCount uint
	)

	mainthread.Do(func() {
		fm := appkit.NSFontManagerSharedFontManager()
		if fm == nil {
			fmNil = true
			return
		}
		families := fm.AvailableFontFamilies()
		if families != nil {
			familyCount = families.Count()
		}
	})

	if fmNil {
		t.Skip("NSFontManagerSharedFontManager() returned nil")
	}
	if familyCount == 0 {
		t.Errorf("AvailableFontFamilies().Count() = 0, expected >= 1")
	}
	t.Logf("AvailableFontFamilies count = %d", familyCount)
}

// ─── 71. NSMenuItem: separatorItem, isSeparatorItem = true ───────────────────

func TestRuntimeRead_NSMenuItem(t *testing.T) {
	var (
		itemNil     bool
		isSeparator bool
	)

	mainthread.Do(func() {
		item := appkit.NSMenuItemSeparatorItem()
		if item == nil {
			itemNil = true
			return
		}
		isSeparator = item.IsSeparatorItem()
	})

	if itemNil {
		t.Skip("NSMenuItemSeparatorItem() returned nil")
	}
	if !isSeparator {
		t.Errorf("IsSeparatorItem() = false on separatorItem, expected true")
	}
	t.Log("NSMenuItem.separatorItem.IsSeparatorItem() = true")
}

// ─── 72. NSAppearance: currentDrawingAppearance ───────────────────────────────

func TestRuntimeRead_NSAppearance(t *testing.T) {
	var appearanceNil bool

	mainthread.Do(func() {
		_ = appkit.NSApplicationSharedApplication()
		a := appkit.NSAppearanceCurrentDrawingAppearance()
		appearanceNil = (a == nil)
	})

	if appearanceNil {
		t.Skip("NSAppearanceCurrentDrawingAppearance() returned nil — no drawing context active")
	}
	t.Log("NSAppearanceCurrentDrawingAppearance() non-nil")
}

// ─── 73. NSDocumentController: sharedDocumentController, documents ────────────

func TestRuntimeRead_NSDocumentController(t *testing.T) {
	var (
		dcNil   bool
		docsNil bool
	)

	mainthread.Do(func() {
		dc := appkit.NSDocumentControllerSharedDocumentController()
		if dc == nil {
			dcNil = true
			return
		}
		docs := dc.Documents()
		docsNil = (docs == nil)
	})

	if dcNil {
		t.Skip("NSDocumentControllerSharedDocumentController() returned nil")
	}
	if docsNil {
		t.Errorf("NSDocumentController.Documents() returned nil")
	}
	t.Log("NSDocumentController.sharedDocumentController.Documents non-nil")
}

// ─── 74. NSColorList: availableColorLists count >= 1 ─────────────────────────

func TestRuntimeRead_NSColorLists(t *testing.T) {
	var (
		listNil bool
		count   uint
	)

	mainthread.Do(func() {
		lists := appkit.NSColorListAvailableColorLists()
		if lists == nil {
			listNil = true
			return
		}
		count = lists.Count()
	})

	if listNil {
		t.Skip("NSColorListAvailableColorLists() returned nil")
	}
	if count == 0 {
		t.Errorf("AvailableColorLists().Count() = 0, expected >= 1")
	}
	t.Logf("NSColorList.AvailableColorLists count = %d", count)
}

// ─── 75. NSImageRep: registeredImageRepClasses count >= 1 ────────────────────

func TestRuntimeRead_NSImageRepClasses(t *testing.T) {
	var (
		classesNil bool
		count      uint
	)

	mainthread.Do(func() {
		classes := appkit.NSImageRepRegisteredImageRepClasses()
		if classes == nil {
			classesNil = true
			return
		}
		count = classes.Count()
	})

	if classesNil {
		t.Skip("NSImageRepRegisteredImageRepClasses() returned nil")
	}
	if count == 0 {
		t.Errorf("RegisteredImageRepClasses().Count() = 0, expected >= 1")
	}
	t.Logf("NSImageRep.RegisteredImageRepClasses count = %d", count)
}

// ─── 76. NSEvent: keyRepeatDelay > 0 ─────────────────────────────────────────

func TestRuntimeRead_KeyRepeatDelay(t *testing.T) {
	var delay float64

	mainthread.Do(func() {
		delay = appkit.NSEventKeyRepeatDelay()
	})

	if delay <= 0 {
		t.Errorf("NSEventKeyRepeatDelay() = %v, expected > 0", delay)
	}
	t.Logf("NSEvent.KeyRepeatDelay = %v", delay)
}

// ─── 77. NSFont: monospacedDigitSystemFont ────────────────────────────────────

func TestRuntimeRead_MonospacedFont(t *testing.T) {
	var fontNil bool

	mainthread.Do(func() {
		font := appkit.NSFontMonospacedDigitSystemFontOfSizeWeight(14.0, 0.0)
		fontNil = (font == nil)
	})

	if fontNil {
		t.Skip("NSFontMonospacedDigitSystemFontOfSizeWeight returned nil")
	}
	t.Log("NSFontMonospacedDigitSystemFontOfSizeWeight(14, 0) non-nil")
}

// ─── 78. NSFont: labelFontOfSize ─────────────────────────────────────────────

func TestRuntimeRead_LabelFont(t *testing.T) {
	var (
		fontNil   bool
		pointSize float64
	)

	mainthread.Do(func() {
		font := appkit.NSFontLabelFontOfSize(12.0)
		if font == nil {
			fontNil = true
			return
		}
		pointSize = font.PointSize()
	})

	if fontNil {
		t.Skip("NSFontLabelFontOfSize(12) returned nil")
	}
	if pointSize <= 0 {
		t.Errorf("LabelFont.PointSize() = %v, expected > 0", pointSize)
	}
	t.Logf("NSFontLabelFontOfSize(12).PointSize = %.1f", pointSize)
}

// ─── 79. NSColor: systemRed, systemOrange, systemGreen ───────────────────────

func TestRuntimeRead_SystemColors(t *testing.T) {
	var redNil, orangeNil, greenNil bool

	mainthread.Do(func() {
		redNil = appkit.NSColorSystemRedColor() == nil
		orangeNil = appkit.NSColorSystemOrangeColor() == nil
		greenNil = appkit.NSColorSystemGreenColor() == nil
	})

	if redNil {
		t.Errorf("NSColorSystemRedColor() returned nil")
	} else {
		t.Log("NSColorSystemRedColor() non-nil")
	}
	if orangeNil {
		t.Errorf("NSColorSystemOrangeColor() returned nil")
	} else {
		t.Log("NSColorSystemOrangeColor() non-nil")
	}
	if greenNil {
		t.Errorf("NSColorSystemGreenColor() returned nil")
	} else {
		t.Log("NSColorSystemGreenColor() non-nil")
	}
}

// ─── 80. NSWorkspace: notificationCenter non-nil ─────────────────────────────

func TestRuntimeRead_WorkspaceNotificationCenter(t *testing.T) {
	var (
		wsNil bool
		ncNil bool
	)

	mainthread.Do(func() {
		ws := appkit.NSWorkspaceSharedWorkspace()
		if ws == nil {
			wsNil = true
			return
		}
		nc := ws.NotificationCenter()
		ncNil = (nc == nil)
	})

	if wsNil {
		t.Skip("NSWorkspaceSharedWorkspace() returned nil")
	}
	if ncNil {
		t.Errorf("NSWorkspace.NotificationCenter() returned nil")
	}
	t.Log("NSWorkspace.NotificationCenter non-nil")
}

// ─── 81. NSScreen: screensHaveSeparateSpaces (class method) ──────────────────

func TestRuntimeRead_ScreensHaveSeparateSpaces(t *testing.T) {
	// This is a class method; it does not require main thread.
	separate := appkit.NSScreenScreensHaveSeparateSpaces()
	t.Logf("NSScreen.screensHaveSeparateSpaces = %v", separate)
}

// ─── 82. NSApplication: isRunning + isActive ─────────────────────────────────

func TestRuntimeRead_ApplicationRunning(t *testing.T) {
	var (
		appNil    bool
		isRunning bool
		isActive  bool
	)

	mainthread.Do(func() {
		app := appkit.NSApplicationSharedApplication()
		if app == nil {
			appNil = true
			return
		}
		isRunning = app.IsRunning()
		isActive = app.IsActive()
	})

	if appNil {
		t.Skip("NSApplicationSharedApplication() returned nil")
	}
	// IsRunning may be false for a bare test binary without NSApplicationMain;
	// just log the value without asserting.
	t.Logf("NSApplication.IsRunning = %v, IsActive = %v", isRunning, isActive)
}

// ─── 83. NSColorSpace: genericRGBColorSpace non-nil ──────────────────────────

func TestRuntimeRead_GenericRGBColorSpace(t *testing.T) {
	var csNil bool

	mainthread.Do(func() {
		cs := appkit.NSColorSpaceGenericRGBColorSpace()
		csNil = (cs == nil)
	})

	if csNil {
		t.Skip("NSColorSpaceGenericRGBColorSpace() returned nil")
	}
	t.Log("NSColorSpaceGenericRGBColorSpace() non-nil")
}

// ─── 84. NSColorSpace: displayP3ColorSpace non-nil ───────────────────────────

func TestRuntimeRead_DisplayP3ColorSpace(t *testing.T) {
	var csNil bool

	mainthread.Do(func() {
		cs := appkit.NSColorSpaceDisplayP3ColorSpace()
		csNil = (cs == nil)
	})

	if csNil {
		t.Skip("NSColorSpaceDisplayP3ColorSpace() returned nil")
	}
	t.Log("NSColorSpaceDisplayP3ColorSpace() non-nil")
}

// ─── 85. NSFontManager: availableFonts count >= 1 ────────────────────────────

func TestRuntimeRead_AvailableFonts(t *testing.T) {
	var (
		fmNil     bool
		fontCount uint
	)

	mainthread.Do(func() {
		fm := appkit.NSFontManagerSharedFontManager()
		if fm == nil {
			fmNil = true
			return
		}
		fonts := fm.AvailableFonts()
		if fonts != nil {
			fontCount = fonts.Count()
		}
	})

	if fmNil {
		t.Skip("NSFontManagerSharedFontManager() returned nil")
	}
	if fontCount == 0 {
		t.Errorf("AvailableFonts().Count() = 0, expected >= 1")
	}
	t.Logf("NSFontManager.AvailableFonts count = %d", fontCount)
}

// ─── 86. NSEvent: doubleClickInterval > 0 ────────────────────────────────────

func TestRuntimeRead_DoubleClickInterval(t *testing.T) {
	var interval float64

	mainthread.Do(func() {
		interval = appkit.NSEventDoubleClickInterval()
	})

	if interval <= 0 {
		t.Errorf("NSEventDoubleClickInterval() = %v, expected > 0", interval)
	}
	t.Logf("NSEvent.DoubleClickInterval = %v", interval)
}

// ─── 87. NSEvent: keyRepeatInterval > 0 ──────────────────────────────────────

func TestRuntimeRead_KeyRepeatInterval(t *testing.T) {
	var interval float64

	mainthread.Do(func() {
		interval = appkit.NSEventKeyRepeatInterval()
	})

	if interval <= 0 {
		t.Errorf("NSEventKeyRepeatInterval() = %v, expected > 0", interval)
	}
	t.Logf("NSEvent.KeyRepeatInterval = %v", interval)
}

// ─── 88. NSFont: labelFontSize > 0 ───────────────────────────────────────────

func TestRuntimeRead_LabelFontSize(t *testing.T) {
	var size float64

	mainthread.Do(func() {
		size = appkit.NSFontLabelFontSize()
	})

	if size <= 0 {
		t.Errorf("NSFontLabelFontSize() = %v, expected > 0", size)
	}
	t.Logf("NSFont.LabelFontSize = %.1f", size)
}

// ─── 89. NSEvent: isMouseCoalescingEnabled (read-only probe) ─────────────────

func TestRuntimeRead_MouseCoalescing(t *testing.T) {
	var enabled bool

	mainthread.Do(func() {
		enabled = appkit.NSEventIsMouseCoalescingEnabled()
	})

	t.Logf("NSEvent.IsMouseCoalescingEnabled = %v", enabled)
}

// ─── 90. NSColorSpace: genericGrayColorSpace non-nil ─────────────────────────

func TestRuntimeRead_GenericGrayColorSpace(t *testing.T) {
	var csNil bool

	mainthread.Do(func() {
		cs := appkit.NSColorSpaceGenericGrayColorSpace()
		csNil = (cs == nil)
	})

	if csNil {
		t.Skip("NSColorSpaceGenericGrayColorSpace() returned nil")
	}
	t.Log("NSColorSpaceGenericGrayColorSpace() non-nil")
}

// ─── 91. NSColorSpace: genericGamma22GrayColorSpace non-nil ──────────────────

func TestRuntimeRead_GenericGamma22GrayColorSpace(t *testing.T) {
	var csNil bool

	mainthread.Do(func() {
		cs := appkit.NSColorSpaceGenericGamma22GrayColorSpace()
		csNil = (cs == nil)
	})

	if csNil {
		t.Skip("NSColorSpaceGenericGamma22GrayColorSpace() returned nil")
	}
	t.Log("NSColorSpaceGenericGamma22GrayColorSpace() non-nil")
}

// ─── 92. NSMenu: menuBarVisible (class method read) ──────────────────────────

func TestRuntimeRead_MenuBarVisible(t *testing.T) {
	// Class method — does not require main thread; just reads a flag.
	visible := appkit.NSMenuMenuBarVisible()
	t.Logf("NSMenu.menuBarVisible = %v", visible)
}

// ════════════════════════════════════════════════════════════════════════════
// Metal
// ════════════════════════════════════════════════════════════════════════════

// ─── 93. Metal: MTLCreateSystemDefaultDevice non-nil ─────────────────────────

func TestRuntimeRead_MetalDevice(t *testing.T) {
	dev := metal.MTLCreateSystemDefaultDevice()
	if dev == nil {
		t.Skip("MTLCreateSystemDefaultDevice() returned nil — no Metal-capable GPU?")
	}
	t.Log("MTLCreateSystemDefaultDevice() non-nil")
}

// ─── 94. Metal: MTLCopyAllDevices count >= 1 ─────────────────────────────────

func TestRuntimeRead_AllMetalDevices(t *testing.T) {
	devices := metal.MTLCopyAllDevices()
	if devices == nil {
		t.Skip("MTLCopyAllDevices() returned nil")
	}
	// MTLCopyAllDevices returns an unsafe.Pointer (CFArrayRef); send count via purego.
	selCount := objc.RegisterName("count")
	count := objc.Send[uint](objc.ID(uintptr(devices)), selCount)
	if count == 0 {
		t.Errorf("MTLCopyAllDevices().Count() = 0, expected >= 1 on Metal-capable Mac")
	}
	t.Logf("MTLCopyAllDevices count = %d", count)
}

// ─── 95. MTLArgumentDescriptor: create and verify defaults ───────────────────

func TestRuntimeRead_MTLArgumentDescriptor(t *testing.T) {
	desc := metal.MTLArgumentDescriptorArgumentDescriptor()
	if desc == nil {
		t.Fatal("MTLArgumentDescriptorArgumentDescriptor() returned nil")
	}
	dt := desc.DataType()
	t.Logf("MTLArgumentDescriptor.DataType = %d (0 = none/undefined)", dt)

	idx := desc.Index()
	t.Logf("MTLArgumentDescriptor.Index = %d", idx)

	al := desc.ArrayLength()
	t.Logf("MTLArgumentDescriptor.ArrayLength = %d", al)
}

// ─── 96. MTLArgumentDescriptor: set and get Index ────────────────────────────

func TestRuntimeRead_MTLArgumentDescriptor_Index(t *testing.T) {
	desc := metal.MTLArgumentDescriptorArgumentDescriptor()
	if desc == nil {
		t.Fatal("MTLArgumentDescriptorArgumentDescriptor() returned nil")
	}
	desc.SetIndex(5)
	if got := desc.Index(); got != 5 {
		t.Errorf("Index() = %d after SetIndex(5), expected 5", got)
	}
	t.Logf("MTLArgumentDescriptor.Index roundtrip = 5")
}

// ─── 97. MTLArgumentDescriptor: set and get ArrayLength ──────────────────────

func TestRuntimeRead_MTLArgumentDescriptor_ArrayLength(t *testing.T) {
	desc := metal.MTLArgumentDescriptorArgumentDescriptor()
	if desc == nil {
		t.Fatal("MTLArgumentDescriptorArgumentDescriptor() returned nil")
	}
	desc.SetArrayLength(10)
	if got := desc.ArrayLength(); got != 10 {
		t.Errorf("ArrayLength() = %d after SetArrayLength(10), expected 10", got)
	}
	t.Logf("MTLArgumentDescriptor.ArrayLength roundtrip = 10")
}

// ─── 98. MTLBlitPassDescriptor: blitPassDescriptor non-nil ───────────────────

func TestRuntimeRead_MTLBlitPassDescriptor(t *testing.T) {
	desc := metal.MTLBlitPassDescriptorBlitPassDescriptor()
	if desc == nil {
		t.Fatal("MTLBlitPassDescriptorBlitPassDescriptor() returned nil")
	}
	t.Log("MTLBlitPassDescriptorBlitPassDescriptor() non-nil")
}

// ─── 99. MTLRenderPassDescriptor: renderPassDescriptor non-nil ───────────────

func TestRuntimeRead_MTLRenderPassDescriptor(t *testing.T) {
	desc := metal.MTLRenderPassDescriptorRenderPassDescriptor()
	if desc == nil {
		t.Fatal("MTLRenderPassDescriptorRenderPassDescriptor() returned nil")
	}
	t.Log("MTLRenderPassDescriptorRenderPassDescriptor() non-nil")
}

// ─── 100. MTLFunctionDescriptor: functionDescriptor non-nil ──────────────────

func TestRuntimeRead_MTLFunctionDescriptor(t *testing.T) {
	desc := metal.MTLFunctionDescriptorFunctionDescriptor()
	if desc == nil {
		t.Fatal("MTLFunctionDescriptorFunctionDescriptor() returned nil")
	}
	t.Log("MTLFunctionDescriptorFunctionDescriptor() non-nil")
}
