package typemap

import (
	"reflect"
	"testing"
)

func TestRealtimeBlockAttributes(t *testing.T) {
	for _, input := range []string{
		"OSStatus (^)(BOOL *, const AudioTimeStamp *, AVAudioFrameCount, AudioBufferList *)",
		"OSStatus (^)(BOOL *, const AudioTimeStamp *, AVAudioFrameCount, AudioBufferList *) __attribute__((nonblocking))",
	} {
		signature, ok := parseBlockComponents(input)
		want := []string{"BOOL *", "const AudioTimeStamp *", "AVAudioFrameCount", "AudioBufferList *"}
		if !ok || signature.ReturnObjCType != "OSStatus" || !reflect.DeepEqual(signature.ParamObjCTypes, want) {
			t.Errorf("parseBlockComponents(%q) = %+v, %v", input, signature, ok)
		}
	}
}

func TestConstScalarAndTypedefValues(t *testing.T) {
	mapper := &Mapper{TypedefIndex: map[string]string{"FixtureValue": "double"}}
	for _, input := range []string{"const double", "double const", "FixtureValue const", "const FixtureValue"} {
		if got := mapper.GoType(input, Context{}, nil); got != "float64" {
			t.Errorf("GoType(%q) = %q, want float64", input, got)
		}
	}
	if got := mapper.GoType("const unsigned char [16]", Context{}, nil); got != "[16]uint8" {
		t.Errorf("const byte array = %q, want [16]uint8", got)
	}
}
