package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dosco/graphjin/v2/core/internal/sdata"
	"github.com/dosco/graphjin/v2/core/internal/util"
)

const (
	KIND_SCALAR      = "SCALAR"
	KIND_OBJECT      = "OBJECT"
	KIND_NONNULL     = "NON_NULL"
	KIND_LIST        = "LIST"
	KIND_UNION       = "UNION"
	KIND_ENUM        = "ENUM"
	KIND_INPUT_OBJ   = "INPUT_OBJECT"
	LOC_QUERY        = "QUERY"
	LOC_MUTATION     = "MUTATION"
	LOC_SUBSCRIPTION = "SUBSCRIPTION"
	LOC_FIELD        = "FIELD"

	SUFFIX_EXP      = "Expression"
	SUFFIX_LISTEXP  = "ListExpression"
	SUFFIX_INPUT    = "Input"
	SUFFIX_ORDER_BY = "OrderByInput"
	SUFFIX_WHERE    = "WhereInput"
	SUFFIX_ARGS     = "ArgsInput"
	SUFFIX_ENUM     = "Enum"
)

var (
	TYPE_STRING  = "String"
	TYPE_INT     = "Int"
	TYPE_BOOLEAN = "Boolean"
	TYPE_FLOAT   = "Float"
	TYPE_JSON    = "JSON"
)

type typeRef struct {
	Kind   string   `json:"kind"`
	Name   *string  `json:"name"`
	OfType *typeRef `json:"ofType"`
}

type inputValue struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Type         *typeRef `json:"type"`
	DefaultValue *string  `json:"defaultValue"`
}

type fieldObj struct {
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Args              []inputValue `json:"args"`
	Type              *typeRef     `json:"type"`
	IsDeprecated      bool         `json:"isDeprecated"`
	DeprecationReason *string      `json:"deprecationReason"`
}

type enumValue struct {
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	IsDeprecated      bool    `json:"isDeprecated"`
	DeprecationReason *string `json:"deprecationReason"`
}

type fullType struct {
	Kind          string       `json:"kind"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Fields        []fieldObj   `json:"fields"`
	InputFields   []inputValue `json:"inputFields"`
	EnumValues    []enumValue  `json:"enumValues"`
	Interfaces    []typeRef    `json:"interfaces"`
	PossibleTypes []typeRef    `json:"possibleTypes"`
}

type shortFullType struct {
	Name string `json:"name"`
}

type directiveType struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Locations    []string     `json:"locations"`
	Args         []inputValue `json:"args"`
	IsRepeatable bool         `json:"isRepeatable"`
}

type introSchema struct {
	Types            []fullType      `json:"types"`
	QueryType        *shortFullType  `json:"queryType"`
	MutationType     *shortFullType  `json:"mutationType"`
	SubscriptionType *shortFullType  `json:"subscriptionType"`
	Directives       []directiveType `json:"directives"`
}

type introResult struct {
	Schema introSchema `json:"__schema"`
}

// const singularSuffix = "ByID"

var stdTypes = []fullType{{
	Kind:        KIND_SCALAR,
	Name:        TYPE_BOOLEAN,
	Description: "The `Boolean` scalar type represents `true` or `false`",
}, {
	Kind:        KIND_SCALAR,
	Name:        TYPE_FLOAT,
	Description: "The `Float` scalar type represents signed double-precision fractional values as specified by\n[IEEE 754](http://en.wikipedia.org/wiki/IEEE_floating_point).",
}, {
	Kind:        KIND_SCALAR,
	Name:        TYPE_INT,
	Description: "The `Int` scalar type represents non-fractional signed whole numeric values. Int can represent\nvalues between -(2^31) and 2^31 - 1.\n",
	// Add Int expression after
}, {
	Kind:        KIND_SCALAR,
	Name:        TYPE_STRING,
	Description: "The `String` scalar type represents textual data, represented as UTF-8 character sequences.\nThe String type is most often used by GraphQL to represent free-form human-readable text.\n",
}, {
	Kind:        KIND_SCALAR,
	Name:        TYPE_JSON,
	Description: "The `JSON` scalar type represents json data",
}, {
	Kind:       KIND_OBJECT,
	Name:       "Query",
	Interfaces: []typeRef{},
	Fields:     []fieldObj{},
}, {
	Kind:       KIND_OBJECT,
	Name:       "Subscription",
	Interfaces: []typeRef{},
	Fields:     []fieldObj{},
}, {
	Kind:       KIND_OBJECT,
	Name:       "Mutation",
	Interfaces: []typeRef{},
	Fields:     []fieldObj{},
}, {
	Kind: KIND_ENUM,
	Name: "FindSearchInput",
	EnumValues: []enumValue{{
		Name:        "children",
		Description: "Children of parent row",
	}, {
		Name:        "parents",
		Description: "Parents of current row",
	}},
}, {
	Kind:        "ENUM",
	Name:        "OrderDirection",
	Description: "Result ordering types",
	EnumValues: []enumValue{{
		Name:        "asc",
		Description: "Ascending order",
	}, {
		Name:        "desc",
		Description: "Descending order",
	}, {
		Name:        "asc_nulls_first",
		Description: "Ascending nulls first order",
	}, {
		Name:        "desc_nulls_first",
		Description: "Descending nulls first order",
	}, {
		Name:        "asc_nulls_last",
		Description: "Ascending nulls last order",
	}, {
		Name:        "desc_nulls_last",
		Description: "Descending nulls last order",
	}},
}, {
	Kind:        KIND_SCALAR,
	Name:        "ID",
	Description: "The `ID` scalar type represents a unique identifier, often used to refetch an object or as key for a cache.\nThe ID type appears in a JSON response as a String; however, it is not intended to be human-readable.\nWhen expected as an input type, any string (such as `\"4\"`) or integer (such as `4`) input value will be accepted\nas an ID.\n",
	// Add IDException after
}, {
	Kind:        KIND_SCALAR,
	Name:        "Cursor",
	Description: "A cursor is an encoded string use for pagination",
},
}

type intro struct {
	schema      *sdata.DBSchema
	camelCase   bool
	types       map[string]fullType
	enumValues  map[string]enumValue
	inputValues map[string]inputValue
	res         introResult
}

func newIntro(schema *sdata.DBSchema, camelCase bool) intro {
	return intro{
		schema:      schema,
		camelCase:   camelCase,
		types:       make(map[string]fullType),
		enumValues:  make(map[string]enumValue),
		inputValues: make(map[string]inputValue),
	}
}

func introspection(in intro) (result json.RawMessage, err error) {
	in.res.Schema = introSchema{
		QueryType:        &shortFullType{Name: "Query"},
		SubscriptionType: &shortFullType{Name: "Subscription"},
		MutationType:     &shortFullType{Name: "Mutation"},
	}

	for _, v := range stdTypes {
		in.types[v.Name] = v
	}

	// Expression types
	v := append(expAll, expScalar...)
	in.addExpTypes(v, "ID", newTR("", "ID", nil))
	in.addExpTypes(v, "String", newTR("", "String", nil))
	in.addExpTypes(v, "Int", newTR("", "Int", nil))
	in.addExpTypes(v, "Boolean", newTR("", "Boolean", nil))
	in.addExpTypes(v, "Float", newTR("", "Float", nil))

	// ListExpression Types
	v = append(expAll, expList...)
	in.addExpTypes(v, "StringList", newTR("", "String", nil))
	in.addExpTypes(v, "IntList", newTR("", "Int", nil))
	in.addExpTypes(v, "BooleanList", newTR("", "Boolean", nil))
	in.addExpTypes(v, "FloatList", newTR("", "Float", nil))

	v = append(expAll, expJSON...)
	in.addExpTypes(v, "JSON", newTR("", "String", nil))

	for alias, t := range in.schema.GetAliases() {
		if err = in.addTable(t, alias); err != nil {
			return
		}
	}

	for _, t := range in.schema.GetTables() {
		if err = in.addTable(t, ""); err != nil {
			return
		}
	}

	for _, dt := range dirTypes {
		in.addDirType(dt)
	}

	for _, v := range in.types {
		in.res.Schema.Types = append(in.res.Schema.Types, v)
	}

	result, err = json.Marshal(in.res)
	return
}

func (in *intro) addTable(t sdata.DBTable, alias string) (err error) {
	if t.Blocked || len(t.Columns) == 0 {
		return
	}
	var ft fullType
	ft, err = in.addTableType(t, alias, 0)
	if err != nil {
		return
	}
	in.addTypeTo("Query", ft)
	in.addTypeTo("Subscription", ft)

	if err = in.addInputType(t, &ft); err != nil {
		return
	}
	in.addTypeTo("Mutation", ft)
	return
}

func (in *intro) addTypeTo(op string, ft fullType) {
	qt := in.types[op]
	qt.Fields = append(qt.Fields, fieldObj{
		Name:        ft.Name,
		Description: ft.Description,
		Args:        ft.InputFields,
		Type:        newTR("", ft.Name, nil),
	})
	in.types[op] = qt
}

func (in *intro) getName(name string) string {
	if in.camelCase {
		return util.ToCamel(name)
	} else {
		return name
	}
}

func (in *intro) addExpTypes(exps []exp, name string, rt *typeRef) {
	tn := (name + SUFFIX_EXP)
	ft := fullType{
		Kind:        KIND_INPUT_OBJ,
		Name:        tn,
		InputFields: []inputValue{},
		Interfaces:  []typeRef{},
	}

	for _, ex := range exps {
		rtVal := rt
		if ex.etype != "" {
			rtVal = newTR("", ex.etype, nil)
		}
		ft.InputFields = append(ft.InputFields, inputValue{
			Name:        ex.name,
			Description: ex.desc,
			Type:        rtVal,
		})
	}
	in.types[tn] = ft
}

func (in *intro) addTableType(t sdata.DBTable, alias string, depth int) (ft fullType, err error) {
	ft = fullType{
		Kind:        KIND_OBJECT,
		InputFields: []inputValue{},
		Interfaces:  []typeRef{},
	}

	name := t.Name
	if alias != "" {
		name = alias
	}
	ft.Name = in.getName(name)
	ft.Description = t.Comment

	var hasSearch bool
	var hasRecursive bool

	if err = in.addColumnsEnumType(t); err != nil {
		return
	}

	for _, fn := range in.schema.GetFunctions() {
		var ty fullType
		ty, err = in.addArgsType(t, fn)
		if err != nil {
			return
		}
		in.types[ty.Name] = ty
	}

	for _, c := range t.Columns {
		if c.Blocked {
			return
		}
		if c.FullText {
			hasSearch = true
		}
		if c.FKRecursive {
			hasRecursive = true
		}
		var f1 fieldObj
		f1, err = in.getColumnField(c)
		if err != nil {
			return
		}
		ft.Fields = append(ft.Fields, f1)
	}

	for _, fn := range in.schema.GetFunctions() {
		var f1 fieldObj
		f1, err = in.getFunctionField(t, fn)
		if err != nil {
			return
		}
		ft.Fields = append(ft.Fields, f1)
	}

	relNodes1, err := in.schema.GetFirstDegree(t)
	if err != nil {
		return
	}

	relNodes2, err := in.schema.GetSecondDegree(t)
	if err != nil {
		return
	}

	for _, relNode := range append(relNodes1, relNodes2...) {
		var f fieldObj
		var skip bool
		f, skip, err = in.getTableField(relNode)
		if err != nil {
			return
		}
		if !skip {
			ft.Fields = append(ft.Fields, f)
		}
	}

	ft.addArg("id", newTR("", "ID", nil))
	ft.addArg("limit", newTR("", "Int", nil))
	ft.addArg("offset", newTR("", "Int", nil))
	ft.addArg("distinctOn", newTR("LIST", "", newTR("", "String", nil)))
	ft.addArg("first", newTR("", "Int", nil))
	ft.addArg("last", newTR("", "Int", nil))
	ft.addArg("after", newTR("", "Cursor", nil))
	ft.addArg("before", newTR("", "Cursor", nil))

	if err = in.addOrderByType(t, &ft); err != nil {
		return
	}

	if err = in.addWhereType(t, &ft); err != nil {
		return
	}

	if err = in.addTableArgsType(t, &ft); err != nil {
		return
	}

	if hasSearch {
		ft.addArg("search", newTR("", "String", nil))
	}

	if depth > 1 {
		return
	}
	if depth > 0 {
		ft.addArg("find", newTR("", "FindSearchInput", nil))
	}

	in.types[name] = ft

	if hasRecursive {
		_, err = in.addTableType(t, (name + "Recursive"), (depth + 1))
	}
	return
}

func (in *intro) addColumnsEnumType(t sdata.DBTable) (err error) {
	tableName := in.getName(t.Name)
	ft := fullType{
		Kind:        KIND_ENUM,
		Name:        (t.Name + SUFFIX_ENUM),
		Description: fmt.Sprintf("Table columns for '%s'", tableName),
	}
	for _, c := range t.Columns {
		if c.Blocked {
			continue
		}
		ft.EnumValues = append(ft.EnumValues, enumValue{
			Name:        in.getName(c.Name),
			Description: c.Comment,
		})
	}
	in.types[ft.Name] = ft
	return
}

func (in *intro) addOrderByType(t sdata.DBTable, ft *fullType) (err error) {
	tn := (t.Name + SUFFIX_ORDER_BY)
	ty := fullType{
		Kind: KIND_INPUT_OBJ,
		Name: tn,
	}
	for _, c := range t.Columns {
		if c.Blocked {
			continue
		}
		ty.InputFields = append(ty.InputFields, inputValue{
			Name:        in.getName(c.Name),
			Description: c.Comment,
			Type:        newTR("", "OrderDirection", nil),
		})
	}
	in.types[tn] = ty
	ft.addArg("orderBy", newTR("", (t.Name+SUFFIX_ORDER_BY), nil))
	return
}

func (in *intro) addWhereType(t sdata.DBTable, ft *fullType) (err error) {
	tn := (t.Name + SUFFIX_WHERE)
	ty := fullType{
		Kind: "INPUT_OBJECT",
		Name: tn,
		InputFields: []inputValue{
			{Name: "and", Type: newTR("", tn, nil)},
			{Name: "or", Type: newTR("", tn, nil)},
			{Name: "not", Type: newTR("", tn, nil)},
		},
	}
	for _, c := range t.Columns {
		if c.Blocked {
			continue
		}
		ft := getTypeFromColumn(c)
		if c.Array {
			ft += SUFFIX_LISTEXP
		} else {
			ft += SUFFIX_EXP
		}
		ty.InputFields = append(ty.InputFields, inputValue{
			Name:        in.getName(c.Name),
			Description: c.Comment,
			Type:        newTR("", ft, nil),
		})
	}
	in.types[ty.Name] = ty
	ft.addArg("where", newTR("", ty.Name, nil))
	return
}

func (in *intro) addInputType(t sdata.DBTable, ft *fullType) (err error) {
	// upsert
	ty := fullType{
		Kind:        "INPUT_OBJECT",
		Name:        ("upsert" + t.Name + SUFFIX_INPUT),
		InputFields: []inputValue{},
	}
	for _, c := range t.Columns {
		if c.Blocked {
			continue
		}
		ft1 := getTypeFromColumn(c)
		ty.InputFields = append(ty.InputFields, inputValue{
			Name:        in.getName(c.Name),
			Description: c.Comment,
			Type:        newTR("", ft1, nil),
		})
	}
	in.types[ty.Name] = ty
	ft.addArg("upsert", newTR("", ty.Name, nil))

	// insert
	relNodes1, err := in.schema.GetFirstDegree(t)
	if err != nil {
		return
	}
	relNodes2, err := in.schema.GetSecondDegree(t)
	if err != nil {
		return
	}
	allNodes := append(relNodes1, relNodes2...)
	fieldLen := len(ty.InputFields)

	ty.Name = ("insert" + t.Name + SUFFIX_INPUT)
	for _, relNode := range allNodes {
		t1 := relNode.Table
		if relNode.Type == sdata.RelRemote ||
			relNode.Type == sdata.RelPolymorphic ||
			relNode.Type == sdata.RelEmbedded {
			continue
		}
		ty.InputFields = append(ty.InputFields, inputValue{
			Name:        in.getName(t1.Name),
			Description: t1.Comment,
			Type:        newTR("", ("insert" + t1.Name + SUFFIX_INPUT), nil),
		})
	}
	in.types[ty.Name] = ty
	ft.addArg("insert", newTR("", ty.Name, nil))

	// update
	ty.Name = ("update" + t.Name + SUFFIX_INPUT)
	i := 0
	for _, relNode := range allNodes {
		t1 := relNode.Table
		if relNode.Type == sdata.RelRemote ||
			relNode.Type == sdata.RelPolymorphic ||
			relNode.Type == sdata.RelEmbedded {
			continue
		}
		ty.InputFields[(fieldLen + i)] = inputValue{
			Name:        in.getName(t1.Name),
			Description: t1.Comment,
			Type:        newTR("", ("update" + t1.Name + SUFFIX_INPUT), nil),
		}
		i++
	}
	desc1 := fmt.Sprintf("Connect to rows in table '%s' that match the expression", in.getName(t.Name))
	ty.InputFields = append(ty.InputFields, inputValue{
		Name:        "connect",
		Description: desc1,
		Type:        newTR("", (t.Name + SUFFIX_WHERE), nil),
	})
	desc2 := fmt.Sprintf("Disconnect from rows in table '%s' that match the expression", in.getName(t.Name))
	ty.InputFields = append(ty.InputFields, inputValue{
		Name:        "disconnect",
		Description: desc2,
		Type:        newTR("", (t.Name + SUFFIX_WHERE), nil),
	})
	desc3 := fmt.Sprintf("Update rows in table '%s' that match the expression", in.getName(t.Name))
	ty.InputFields = append(ty.InputFields, inputValue{
		Name:        "where",
		Description: desc3,
		Type:        newTR("", (t.Name + SUFFIX_WHERE), nil),
	})
	in.types[ty.Name] = ty
	ft.addArg("update", newTR("", ty.Name, nil))

	// delete
	ft.addArg("delete", newTR("", TYPE_BOOLEAN, nil))
	return
}

func (in *intro) addTableArgsType(t sdata.DBTable, ft *fullType) (err error) {
	if t.Type != "function" {
		return
	}
	ty, err := in.addArgsType(t, t.Func)
	if err != nil {
		return
	}
	in.types[ty.Name] = ty
	ft.addArg("args", newTR("", ty.Name, nil))
	return
}

func (in *intro) addArgsType(t sdata.DBTable, fn sdata.DBFunction) (ft fullType, err error) {
	ft = fullType{
		Kind: "INPUT_OBJECT",
		Name: (t.Name + fn.Name + SUFFIX_ARGS),
	}
	for _, fi := range fn.Inputs {
		var tr *typeRef
		if fn.Agg {
			tr = newTR("", (t.Name + SUFFIX_ENUM), nil)
		} else {
			tn, list := getType(fi.Type)
			if tn == "" {
				tn = "String"
			}
			tr = newTR("", tn, nil)
			if list {
				tr = newTR(KIND_LIST, "", tr)
			}
		}

		fname := in.getName(fi.Name)
		if fname == "" {
			fname = "_" + strconv.Itoa(fi.ID)
		}
		ft.InputFields = append(ft.InputFields, inputValue{
			Name: fname,
			Type: tr,
		})
	}
	return
}

func (in *intro) getColumnField(c sdata.DBColumn) (f fieldObj, err error) {
	f.Args = []inputValue{}
	f.Name = in.getName(c.Name)
	typeValue := newTR("", "String", nil)

	if v, ok := in.types[getTypeFromColumn(c)]; ok {
		typeValue.Name = &v.Name
		typeValue.Kind = v.Kind
	}

	if c.Array {
		typeValue = newTR(KIND_LIST, "", typeValue)
	}

	if c.NotNull {
		typeValue = newTR(KIND_NONNULL, "", typeValue)
	}

	f.Type = typeValue
	return
}

func (in *intro) getFunctionField(t sdata.DBTable, fn sdata.DBFunction) (f fieldObj, err error) {
	f.Name = in.getName(fn.Name)
	f.Args = []inputValue{}
	ty, list := getType(fn.Type)
	f.Type = newTR("", ty, nil)
	if list {
		f.Type = newTR(KIND_LIST, "", f.Type)
	}

	if len(fn.Inputs) != 0 {
		typeName := (t.Name + fn.Name + SUFFIX_ARGS)
		argsArg := inputValue{Name: "args", Type: newTR("", typeName, nil)}
		f.Args = append(f.Args, argsArg)
	}
	return
}

func (in *intro) getTableField(relNode sdata.RelNode) (
	f fieldObj, skip bool, err error) {
	f.Args = []inputValue{}
	f.Name = in.getName(relNode.Name)

	tn := relNode.Table.Name
	if _, ok := in.types[tn]; !ok && relNode.Type != sdata.RelRecursive {
		skip = true
		return
	}

	switch relNode.Type {
	case sdata.RelOneToOne:
		f.Type = newTR(KIND_LIST, "", newTR("", tn, nil))
	case sdata.RelRecursive:
		tn += "Recursive"
		f.Type = newTR(KIND_LIST, "", newTR("", tn, nil))
	default:
		f.Type = newTR("", tn, nil)
	}
	return
}

func (in *intro) addDirType(dt dir) {
	d := directiveType{
		Name:         dt.name,
		Description:  dt.desc,
		Locations:    dt.locs,
		IsRepeatable: dt.repeat,
	}
	for _, a := range dt.args {
		d.Args = append(d.Args, inputValue{
			Name:        a.name,
			Description: a.desc,
			Type:        newTR("", a.atype, nil),
		})
	}
	if len(dt.args) == 0 {
		d.Args = []inputValue{}
	}
	in.res.Schema.Directives = append(in.res.Schema.Directives, d)
}

func (ft *fullType) addArg(name string, tr *typeRef) {
	ft.InputFields = append(ft.InputFields, inputValue{
		Name: name,
		Type: tr,
	})
}

func newTR(kind, name string, tr *typeRef) *typeRef {
	if name == "" {
		return &typeRef{Kind: kind, Name: nil, OfType: tr}
	}
	return &typeRef{Kind: kind, Name: &name, OfType: tr}
}

func getTypeFromColumn(col sdata.DBColumn) (gqlType string) {
	if col.PrimaryKey {
		gqlType = "ID"
		return
	}
	gqlType, _ = getType(col.Type)
	return
}

func getType(t string) (gqlType string, list bool) {
	if i := strings.IndexRune(t, '('); i != -1 {
		t = t[:i]
	}
	if i := strings.IndexRune(t, '['); i != -1 {
		list = true
		t = t[:i]
	}
	if v, ok := dbTypes[t]; ok {
		gqlType = v
	} else if t == "json" || t == "jsonb" {
		gqlType = "JSON"
	} else {
		gqlType = "String"
	}
	return
}
