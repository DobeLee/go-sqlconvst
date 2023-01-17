package main

import (
	"flag"
	"fmt"
	"github.com/gookit/goutil/strutil"
	"github.com/xwb1989/sqlparser"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
	"strings"
)

var (
	sqlPath string
	cfgPath string
	daemon  bool
	help    bool

	cfg *Config
	ddl *sqlparser.DDL
)

type Config struct {
	Source struct {
		ConnectDrive  string `yaml:"connect_drive"`
		ConnectString string `yaml:"connect_string"`
		Database      string `yaml:"database"`
		Table         string `yaml:"table"`
	} `yaml:"source"`
	TableLower    bool `yaml:"table_lower"`
	ColumnLower   bool `yaml:"column_lower"`
	IgnoreComment bool `yaml:"ignore_comment"`
	BoolType      bool `yaml:"bool_type"`
	TagJson       bool `yaml:"tag_json"`
	TagDB         bool `yaml:"tag_db"`
	ForceUnsigned bool `yaml:"force_unsigned"`
	TagGorm       struct {
		Enable          bool `yaml:"enable"`
		GormColumn      bool `yaml:"gorm_column"`
		GormCurrentTime bool `yaml:"gorm_current_time"`
		GormPrimaryKey  bool `yaml:"gorm_primary_key"`
	} `yaml:"tag_gorm"`
}

const (
	CHAR       = "char"
	VARCHAR    = "varchar"
	BINARY     = "binary"
	VARBINARY  = "varbinary"
	TINYBLOB   = "tinyblob"
	TINYTEXT   = "tinytext"
	TEXT       = "text"
	BLOB       = "blob"
	MEDIUMTEXT = "mediumtext"
	MEDIUMBLOB = "mediumblob"
	LONGTEXT   = "longtext"
	LONGBLOB   = "longblob"
	BIT        = "bit"
	TINYINT    = "tinyint"
	BOOL       = "bool"
	BOOLEAN    = "boolean"
	SMALLINT   = "smallint"
	MEDIUMINT  = "mediumint"
	INT        = "int"
	INTEGER    = "integer"
	BIGINT     = "bigint"
	FLOAT      = "float"
	DOUBLE     = "double"
	DECIMAL    = "decimal"
	DEC        = "dec"
	DATE       = "date"
	DATETIME   = "datetime"
	TIMESTAMP  = "timestamp"
	TIME       = "time"
	YEAR       = "year"
)

const (
	SQLCurTime = "current_timestamp"
)

const (
	GoTIME    = "time.Time"
	GoSTRING  = "string"
	GoINT8    = "int8"
	GoINT16   = "int16"
	GoINT32   = "int32"
	GoINT64   = "int64"
	GoFLOAT32 = "float32"
	GoFLOAT64 = "float64"
	GoBOOL    = "bool"
	GoBYTES   = "[]byte"
	GoBIT     = "[]uint8"
)

type Table struct {
	Name      string
	GoName    string
	Comment   string
	GoComment string
	Cols      []Col
}

type Col struct {
	// sql
	Name         string
	TypeName     string
	TypeLen      string
	TypeLenInt   int
	Comment      string
	Unsigned     bool
	IsPrimary    bool
	DefaultValue string
	OnUpdate     string
	// go
	GoName          string
	GoType          string
	GoComment       string
	GoTags          []string
	GormSubTags     []string
	GormSubTagsText string
	// type
	IsString   bool
	IsNumeric  bool
	IsDateTime bool
	IsBool     bool
}

const (
	TplTagJson    = `json:"%s"`
	TplTagDb      = `db:"%s"`
	TplTagGorm    = `gorm:"%s"`
	TplTagGormPK  = `primaryKey`
	TplTagGormCOL = `column:%s`
	TplTagGormDT  = `default:current_time`
)

func main() {

	flag.StringVar(&sqlPath, "s", "my.sql", "The file include CREATE SQL")
	flag.StringVar(&cfgPath, "c", "default.yaml", "Config")
	flag.BoolVar(&daemon, "d", false, "Daemon")
	flag.BoolVar(&help, "h", false, "Show help")
	flag.Parse()

	if help {
		flag.PrintDefaults()
		return
	}

	cfgFileBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		panic(err)
	}

	cfg = new(Config)
	err = yaml.Unmarshal(cfgFileBytes, cfg)
	if err != nil {
		panic(err)
	}

	sqlFileBytes, err := os.ReadFile(sqlPath)
	if err != nil {
		panic(err)
	}

	stmt, err := sqlparser.Parse(string(sqlFileBytes))
	if err != nil {
		panic("This may not be a SQL")
	}

	switch stmt := stmt.(type) {
	case *sqlparser.DDL:
		ddl = stmt
	default:
		panic("Only support DDL CREATE SQL")
	}

	t := Table{
		Name: ddl.NewName.Name.String(),
	}
	t.SetTableGoName(ddl.NewName.Name.String())
	cols := make([]Col, 0)
	primaries := make(map[string]struct{}, 0)

	if ddl.TableSpec != nil {
		for _, index := range ddl.TableSpec.Indexes {
			if index.Info != nil {
				if index.Info.Primary {
					for _, pCol := range index.Columns {
						primaries[pCol.Column.String()] = struct{}{}
					}
				}
			}
		}
	}

	for _, ddlCol := range ddl.TableSpec.Columns {
		colName := ddlCol.Name.String()
		col := Col{
			Name:     colName,
			TypeName: ddlCol.Type.Type,
			Unsigned: bool(ddlCol.Type.Unsigned),
		}
		col.GoName = col.SetColGoName(colName)
		if _, ok := primaries[colName]; ok {
			col.IsPrimary = true
		}
		if ddlCol.Type.Length != nil {
			col.TypeLen = sqlparser.String(ddlCol.Type.Length)
			col.TypeLenInt, _ = strconv.Atoi(col.TypeLen)
		}
		if ddlCol.Type.Comment != nil {
			col.Comment = sqlparser.String(ddlCol.Type.Comment)
		}
		if ddlCol.Type.Default != nil {
			col.DefaultValue = sqlparser.String(ddlCol.Type.Default)
		}
		if ddlCol.Type.OnUpdate != nil {
			col.OnUpdate = sqlparser.String(ddlCol.Type.OnUpdate)
		}
		col.ColDDL2GO()
		cols = append(cols, col)
	}
	t.Cols = cols

	fmt.Println(t.Generate())
}

func (c *Col) ColDDL2GO() {
	c.SetGoType()
	c.SetGoComment()
	if cfg.TagJson {
		c.SetGoJsonTag()
	}
	if cfg.TagDB {
		c.SetGoDBTag()
	}
	if cfg.TagGorm.Enable {
		c.SetGoGormTag()
	}
	c.Generate()
}

func (c *Col) SetGoType() {
	c.TypeDDL2GO()
}

func (c *Col) SetGoComment() {
	c.GoComment = strings.Trim(c.Comment, "'")
}

func (c *Col) SetGoJsonTag() {
	c.GoTags = append(c.GoTags, fmt.Sprintf(TplTagJson, c.Name))
}
func (c *Col) SetGoDBTag() {
	c.GoTags = append(c.GoTags, fmt.Sprintf(TplTagDb, c.Name))
}

func (c *Col) SetGoGormTag() {
	gormSubTags := make([]string, 0)
	if cfg.TagGorm.GormColumn {
		gormSubTags = append(gormSubTags, fmt.Sprintf(TplTagGormCOL, c.Name))
	}
	if c.IsPrimary && cfg.TagGorm.GormPrimaryKey {
		gormSubTags = append(gormSubTags, TplTagGormPK)
	}
	if c.IsDateTime && c.DefaultValue == SQLCurTime && cfg.TagGorm.GormCurrentTime {
		gormSubTags = append(gormSubTags, TplTagGormDT)
	}
	c.GoTags = append(c.GoTags, fmt.Sprintf(TplTagGorm, strings.Join(gormSubTags, ";")))
}

func (c *Col) TypeDDL2GO() {
	typeName := strings.ToLower(c.TypeName)
	switch typeName {
	case DATE, DATETIME, YEAR, TIMESTAMP, TIME:
		c.SetTime(GoTIME)
	case CHAR, VARCHAR, VARBINARY, TINYTEXT, TEXT, MEDIUMTEXT, LONGTEXT:
		c.SetString(GoSTRING)
	case TINYBLOB, MEDIUMBLOB, BLOB, LONGBLOB, BINARY:
		c.SetType(GoBYTES)
	case BOOL, BOOLEAN:
		if cfg.BoolType {
			c.SetBool(GoBOOL)
		} else {
			c.SetInt(GoINT8, true)
		}
	case BIT:
		c.SetType(GoBIT)
	case TINYINT:
		if c.TypeLenInt == 1 && cfg.BoolType {
			c.SetBool(GoBOOL)
		} else {
			c.SetInt(GoINT8, c.Unsigned)
		}
	case SMALLINT:
		c.SetInt(GoINT16, c.Unsigned)
	case MEDIUMINT, INT, INTEGER:
		c.SetInt(GoINT32, c.Unsigned)
	case BIGINT:
		c.SetInt(GoINT64, c.Unsigned)
	case FLOAT:
		c.SetFloat(GoFLOAT32)
	case DOUBLE, DECIMAL, DEC:
		c.SetFloat(GoFLOAT64)
	default:
		c.SetString(GoSTRING)
	}
}

func (c *Col) SetInt(goType string, us bool) {
	c.IsNumeric = true
	if us || cfg.ForceUnsigned {
		goType = fmt.Sprintf("u%s", goType)
	}
	c.GoType = goType
}

func (c *Col) SetFloat(goType string) {
	c.IsNumeric = true
	c.GoType = goType
}

func (c *Col) SetString(goType string) {
	c.IsString = true
	c.GoType = goType
}

func (c *Col) SetTime(goType string) {
	c.IsDateTime = true
	c.GoType = goType
}

func (c *Col) SetBool(goType string) {
	c.IsBool = true
	c.GoType = goType
}

func (c *Col) SetType(goType string) {
	c.GoType = goType
}

func (c *Col) SetColGoName(n string) string {
	if cfg.ColumnLower {
		return n
	}
	return goName(n)
}

func (c *Col) Generate() {
	if len(c.GoTags) > 0 {
		c.GormSubTagsText = fmt.Sprintf("`%s`", strings.Join(c.GoTags, " "))
	}
}

func (t *Table) Generate() string {
	tText := fmt.Sprintf("type %s struct {\n", goName(ddl.NewName.Name.String()))
	var nameMaxLen, typeMaxLen, tagMaxLen int
	for _, col := range t.Cols {
		nLen := strutil.Utf8Len(col.GoName)
		tpLen := strutil.Utf8Len(col.GoType)
		tgLen := strutil.Utf8Len(col.GormSubTagsText)
		if nLen > nameMaxLen {
			nameMaxLen = nLen
		}
		if tpLen > typeMaxLen {
			typeMaxLen = tpLen
		}
		if tgLen > tagMaxLen {
			tagMaxLen = tgLen
		}
	}
	for _, col := range t.Cols {
		tText += fmt.Sprintf(strings.Repeat(" ", 4)+"%-"+strconv.Itoa(nameMaxLen)+"s ", col.GoName)
		tText += fmt.Sprintf("%-"+strconv.Itoa(typeMaxLen)+"s ", col.GoType)
		if tagMaxLen > 0 {
			tText += fmt.Sprintf("%-"+strconv.Itoa(tagMaxLen)+"s ", col.GormSubTagsText)
		}
		if strutil.Utf8Len(col.GoComment) > 0 && !cfg.IgnoreComment {
			tText += fmt.Sprintf("// %s", col.GoComment)
		}
		tText += "\n"
	}
	tText += "}\n"
	return tText
}

func (t *Table) SetTableGoName(n string) string {
	if cfg.TableLower {
		return n
	}
	return goName(n)
}

func goName(sqlName string) string {
	return strings.ReplaceAll(strutil.UpperWord(sqlName), "_", "")
}
