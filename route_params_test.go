package golly

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

type paramsAllRequired struct {
	WorkroomID string `json:"workroom_id" required:"true"`
	Name       string `json:"name"        required:"true"`
}

type paramsAllOptional struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

type paramsMixed struct {
	ID    string `json:"id"    validate:"required"`
	Notes string `json:"notes"`
}

type paramsValidateTag struct {
	Stage string `json:"stage" validate:"required,oneof=open closed"`
	Notes string `json:"notes"`
}

type paramsNoJSON struct {
	WorkroomID string
	Name       string `required:"true"`
}

type paramsExcluded struct {
	Keep    string `json:"keep"`
	Skipped string `json:"-"`
	hidden  string // unexported
}

type paramsPointer struct {
	ID string `json:"id" required:"true"`
}

// ---------------------------------------------------------------------------
// Params[T]
// ---------------------------------------------------------------------------

func TestParams_AllRequired(t *testing.T) {
	ps := Params[paramsAllRequired]()

	require.Len(t, ps, 2)

	assert.Equal(t, "workroom_id", ps[0].Name)
	assert.True(t, ps[0].Required)
	assert.Equal(t, ParamSourceBody, ps[0].Source)

	assert.Equal(t, "name", ps[1].Name)
	assert.True(t, ps[1].Required)
}

func TestParams_AllOptional(t *testing.T) {
	ps := Params[paramsAllOptional]()

	require.Len(t, ps, 2)
	assert.Equal(t, "cursor", ps[0].Name)
	assert.False(t, ps[0].Required)

	assert.Equal(t, "limit", ps[1].Name)
	assert.False(t, ps[1].Required)
}

func TestParams_Mixed(t *testing.T) {
	ps := Params[paramsMixed]()

	require.Len(t, ps, 2)

	assert.Equal(t, "id", ps[0].Name)
	assert.True(t, ps[0].Required, "validate:required should mark field required")

	assert.Equal(t, "notes", ps[1].Name)
	assert.False(t, ps[1].Required)
}

func TestParams_ValidateTagWithOptions(t *testing.T) {
	ps := Params[paramsValidateTag]()

	require.Len(t, ps, 2)
	assert.Equal(t, "stage", ps[0].Name)
	assert.True(t, ps[0].Required, "validate:required,oneof=... should still be required")
}

func TestParams_NoJSONTag_FallsBackToLowercase(t *testing.T) {
	ps := Params[paramsNoJSON]()

	require.Len(t, ps, 2)
	assert.Equal(t, "workroomid", ps[0].Name)
	assert.False(t, ps[0].Required)

	assert.Equal(t, "name", ps[1].Name)
	assert.True(t, ps[1].Required)
}

func TestParams_ExcludesDashedAndUnexported(t *testing.T) {
	ps := Params[paramsExcluded]()

	require.Len(t, ps, 1, "json:\"-\" and unexported fields should be excluded")
	assert.Equal(t, "keep", ps[0].Name)
}

func TestParams_Pointer(t *testing.T) {
	ps := Params[*paramsPointer]()

	require.Len(t, ps, 1)
	assert.Equal(t, "id", ps[0].Name)
	assert.True(t, ps[0].Required)
}

func TestParams_NonStruct_ReturnsNil(t *testing.T) {
	ps := Params[string]()
	assert.Nil(t, ps)

	ps2 := Params[int]()
	assert.Nil(t, ps2)
}

func TestParams_TypeField(t *testing.T) {
	ps := Params[paramsAllOptional]()

	require.Len(t, ps, 2)
	assert.Equal(t, "string", ps[0].Type)
	assert.Equal(t, "int", ps[1].Type)
}

// ---------------------------------------------------------------------------
// formatRouteParams
// ---------------------------------------------------------------------------

func TestFormatRouteParams_Empty(t *testing.T) {
	assert.Equal(t, "", formatRouteParams(nil))
	assert.Equal(t, "", formatRouteParams(RouteParamSet{}))
}

func TestFormatRouteParams_RequiredAndOptional(t *testing.T) {
	ps := RouteParamSet{
		{Name: "id", Required: true},
		{Name: "notes", Required: false},
	}
	assert.Equal(t, " [id*, notes?]", formatRouteParams(ps))
}

func TestFormatRouteParams_AllRequired(t *testing.T) {
	ps := RouteParamSet{
		{Name: "stage", Required: true},
		{Name: "name", Required: true},
	}
	assert.Equal(t, " [stage*, name*]", formatRouteParams(ps))
}

func TestFormatRouteParams_AllOptional(t *testing.T) {
	ps := RouteParamSet{
		{Name: "cursor", Required: false},
		{Name: "limit", Required: false},
	}
	assert.Equal(t, " [cursor?, limit?]", formatRouteParams(ps))
}

// ---------------------------------------------------------------------------
// Integration: params stored on Route node
// ---------------------------------------------------------------------------

func TestRouteParams_StoredOnNode(t *testing.T) {
	root := NewRouteRoot()
	root.Post("/create", noOpHandler, Params[paramsAllRequired]())

	node := FindRoute(root, "/create")
	require.NotNil(t, node)

	idx := methodIndex(POST)
	ps := node.params[idx]

	require.Len(t, ps, 2)
	assert.Equal(t, "workroom_id", ps[0].Name)
	assert.True(t, ps[0].Required)
}

func TestRouteParams_DifferentMethodsDifferentParams(t *testing.T) {
	root := NewRouteRoot()
	root.Post("/resource", noOpHandler, Params[paramsAllRequired]())
	root.Put("/resource", noOpHandler, Params[paramsAllOptional]())

	node := FindRoute(root, "/resource")
	require.NotNil(t, node)

	postParams := node.params[methodIndex(POST)]
	putParams := node.params[methodIndex(PUT)]

	assert.Len(t, postParams, 2)
	assert.Len(t, putParams, 2)
	assert.Equal(t, "workroom_id", postParams[0].Name)
	assert.Equal(t, "cursor", putParams[0].Name)
}

func TestRouteParams_NoParamsLeavesSliceNil(t *testing.T) {
	root := NewRouteRoot()
	root.Get("/ping", noOpHandler)

	node := FindRoute(root, "/ping")
	require.NotNil(t, node)

	assert.Nil(t, node.params[methodIndex(GET)])
}

func TestRouteParams_BackwardCompatible_NoParamArg(t *testing.T) {
	// All method helpers must work without the params arg — no panics.
	root := NewRouteRoot()
	assert.NotPanics(t, func() {
		root.Get("/a", noOpHandler)
		root.Post("/b", noOpHandler)
		root.Put("/c", noOpHandler)
		root.Patch("/d", noOpHandler)
		root.Delete("/e", noOpHandler)
		root.Options("/f", noOpHandler)
		root.Connect("/g", noOpHandler)
		root.Head("/h", noOpHandler)
	})
}

// ---------------------------------------------------------------------------
// Integration: buildPath output includes param hints
// ---------------------------------------------------------------------------

func TestBuildPath_WithParams(t *testing.T) {
	root := NewRouteRoot()
	root.Post("/create", noOpHandler, Params[paramsAllRequired]())

	lines := buildPath(root, "")

	// Should contain the POST line with param annotation
	require.Len(t, lines, 1)
	assert.Equal(t, "[POST] /create [workroom_id*, name*]", lines[0])
}

func TestBuildPath_NoParams_NoAnnotation(t *testing.T) {
	root := NewRouteRoot()
	root.Get("/ping", noOpHandler)

	lines := buildPath(root, "")

	require.Len(t, lines, 1)
	assert.Equal(t, "[GET] /ping", lines[0])
}

func TestBuildPath_MixedParamsAndNone(t *testing.T) {
	root := NewRouteRoot()
	root.Get("/list", noOpHandler)
	root.Post("/create", noOpHandler, Params[paramsMixed]())

	lines := buildPath(root, "")
	sort.Strings(lines)

	assert.Equal(t, []string{
		"[GET] /list",
		"[POST] /create [id*, notes?]",
	}, lines)
}
