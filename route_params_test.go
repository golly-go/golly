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
// RouteDoc Tests
// ---------------------------------------------------------------------------

func TestParams_AllRequired(t *testing.T) {
	ps := Body(paramsAllRequired{}).params

	require.Len(t, ps, 2)

	assert.Equal(t, "workroom_id", ps[0].Name)
	assert.True(t, ps[0].Required)
	assert.Equal(t, ParamSourceBody, ps[0].Source)

	assert.Equal(t, "name", ps[1].Name)
	assert.True(t, ps[1].Required)
}

func TestParams_AllOptional(t *testing.T) {
	ps := Body(paramsAllOptional{}).params

	require.Len(t, ps, 2)
	assert.Equal(t, "cursor", ps[0].Name)
	assert.False(t, ps[0].Required)

	assert.Equal(t, "limit", ps[1].Name)
	assert.False(t, ps[1].Required)
}

func TestParams_Mixed(t *testing.T) {
	ps := Body(paramsMixed{}).params

	require.Len(t, ps, 2)

	assert.Equal(t, "id", ps[0].Name)
	assert.True(t, ps[0].Required, "validate:required should mark field required")

	assert.Equal(t, "notes", ps[1].Name)
	assert.False(t, ps[1].Required)
}

func TestParams_ValidateTagWithOptions(t *testing.T) {
	ps := Body(paramsValidateTag{}).params

	require.Len(t, ps, 2)
	assert.Equal(t, "stage", ps[0].Name)
	assert.True(t, ps[0].Required, "validate:required,oneof=... should still be required")
}

func TestParams_NoJSONTag_FallsBackToLowercase(t *testing.T) {
	ps := Body(paramsNoJSON{}).params

	require.Len(t, ps, 2)
	assert.Equal(t, "workroomid", ps[0].Name)
	assert.False(t, ps[0].Required)

	assert.Equal(t, "name", ps[1].Name)
	assert.True(t, ps[1].Required)
}

func TestParams_ExcludesDashedAndUnexported(t *testing.T) {
	ps := Body(paramsExcluded{}).params

	require.Len(t, ps, 1, "json:\"-\" and unexported fields should be excluded")
	assert.Equal(t, "keep", ps[0].Name)
}

func TestParams_Pointer(t *testing.T) {
	ps := Body(&paramsPointer{}).params

	require.Len(t, ps, 1)
	assert.Equal(t, "id", ps[0].Name)
	assert.True(t, ps[0].Required)
}

func TestParams_NonStruct_ReturnsNil(t *testing.T) {
	ps := Body("").params
	assert.Nil(t, ps)

	ps2 := Body(1).params
	assert.Nil(t, ps2)
}

func TestParams_TypeField(t *testing.T) {
	ps := Body(paramsAllOptional{}).params

	require.Len(t, ps, 2)
	assert.Equal(t, "string", ps[0].Type)
	assert.Equal(t, "int", ps[1].Type)
}

// ---------------------------------------------------------------------------
// formatRouteDoc
// ---------------------------------------------------------------------------

func TestFormatRouteDoc_Empty(t *testing.T) {
	assert.Equal(t, "", formatRouteDoc(nil))
	assert.Equal(t, "", formatRouteDoc(&RouteDoc{}))
}

func TestFormatRouteDoc_RequiredAndOptional(t *testing.T) {
	doc := Describe("My doc").Body(struct {
		ID    string `json:"id" required:"true"`
		Notes string `json:"notes"`
	}{})
	assert.Equal(t, " [id: string*, notes: string?]\t\"My doc\"", formatRouteDoc(doc))
}

func TestFormatRouteDoc_AllRequired(t *testing.T) {
	doc := Body(struct {
		Stage string `json:"stage" required:"true"`
		Name  string `json:"name" required:"true"`
	}{})
	assert.Equal(t, " [stage: string*, name: string*]", formatRouteDoc(doc))
}

func TestFormatRouteDoc_AllOptional(t *testing.T) {
	doc := Describe("Docs").Body(struct {
		Cursor string `json:"cursor"`
		Limit  int    `json:"limit"`
	}{})
	assert.Equal(t, " [cursor: string?, limit: int?]\t\"Docs\"", formatRouteDoc(doc))
}

// ---------------------------------------------------------------------------
// Integration: params stored on Route node
// ---------------------------------------------------------------------------

func TestRouteParams_StoredOnNode(t *testing.T) {
	root := NewRouteRoot()
	root.Post("/create", noOpHandler, Body(paramsAllRequired{}))

	node := FindRoute(root, "/create")
	require.NotNil(t, node)

	idx := methodIndex(POST)
	doc := node.docs[idx]

	require.NotNil(t, doc)
	require.Len(t, doc.params, 2)
	assert.Equal(t, "workroom_id", doc.params[0].Name)
	assert.True(t, doc.params[0].Required)
}

func TestRouteParams_DifferentMethodsDifferentParams(t *testing.T) {
	root := NewRouteRoot()
	root.Post("/resource", noOpHandler, Body(paramsAllRequired{}))
	root.Put("/resource", noOpHandler, Body(paramsAllOptional{}))

	node := FindRoute(root, "/resource")
	require.NotNil(t, node)

	postDoc := node.docs[methodIndex(POST)]
	putDoc := node.docs[methodIndex(PUT)]

	require.NotNil(t, postDoc)
	require.NotNil(t, putDoc)

	assert.Len(t, postDoc.params, 2)
	assert.Len(t, putDoc.params, 2)
	assert.Equal(t, "workroom_id", postDoc.params[0].Name)
	assert.Equal(t, "cursor", putDoc.params[0].Name)
}

func TestRouteParams_NoParamsLeavesSliceNil(t *testing.T) {
	root := NewRouteRoot()
	root.Get("/ping", noOpHandler)

	node := FindRoute(root, "/ping")
	require.NotNil(t, node)

	assert.Nil(t, node.docs[methodIndex(GET)])
}

func TestRouteParams_BackwardCompatible_NoParamArg(t *testing.T) {
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
	root.Post("/create", noOpHandler, Describe("Test").Body(paramsAllRequired{}))

	lines := buildPath(root, "")

	require.Len(t, lines, 1)
	assert.Equal(t, "[POST]\t/create\t [workroom_id: string*, name: string*]\t\"Test\"", lines[0])
}

func TestBuildPath_NoParams_NoAnnotation(t *testing.T) {
	root := NewRouteRoot()
	root.Get("/ping", noOpHandler)

	lines := buildPath(root, "")

	require.Len(t, lines, 1)
	assert.Equal(t, "[GET]\t/ping\t", lines[0])
}

func TestBuildPath_MixedParamsAndNone(t *testing.T) {
	root := NewRouteRoot()
	root.Get("/list", noOpHandler)
	root.Post("/create", noOpHandler, Body(paramsMixed{}))

	lines := buildPath(root, "")
	sort.Strings(lines)

	assert.Equal(t, []string{
		"[GET]\t/list\t",
		"[POST]\t/create\t [id: string*, notes: string?]",
	}, lines)
}
