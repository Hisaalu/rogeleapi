package crud

import (
	"fmt"
	"strings"

	"github.com/Hisaalu/rogele/internal/domain"
)

type ColumnType string

const (
	TypeString  ColumnType = "string"
	TypeInteger ColumnType = "integer"
	TypeNumber  ColumnType = "number"
	TypeBoolean ColumnType = "boolean"
)

var builtinSystemColumns = []string{
	"id",
	"created_at",
	"updated_at",
}

type ColumnDef struct {
	Type     ColumnType `yaml:"type"`
	Writable bool       `yaml:"writable"`
	Required bool       `yaml:"required"`
}

type Resource struct {
	Name          string               `yaml:"name"`
	Table         string               `yaml:"table"`
	PrimaryKey    string               `yaml:"primary_key"`
	Searchable    []string             `yaml:"searchable"`
	Filterable    []string             `yaml:"filterable"`
	DefaultSort   []string             `yaml:"default_sort"`
	ReadRoles     []string             `yaml:"read_roles"`
	WriteRoles    []string             `yaml:"write_roles"`
	SystemColumns []string             `yaml:"system_columns"`
	Columns       map[string]ColumnDef `yaml:"columns"`

	readSet  map[domain.UserRole]struct{}
	writeSet map[domain.UserRole]struct{}
	known    map[string]struct{}
}

type Config struct {
	Resources []*Resource `yaml:"resources"`
}

func (r *Resource) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("resource: name is required")
	}
	if r.Table == "" {
		return fmt.Errorf("resource %s: table is required", r.Name)
	}
	if r.PrimaryKey == "" {
		r.PrimaryKey = "id"
	}
	if !isValidIdent(r.Table) {
		return fmt.Errorf("resource %s: invalid table name %q", r.Name, r.Table)
	}
	if !isValidIdent(r.PrimaryKey) {
		return fmt.Errorf("resource %s: invalid primary key %q", r.Name, r.PrimaryKey)
	}
	if len(r.Columns) == 0 {
		return fmt.Errorf("resource %s: at least one column must be declared", r.Name)
	}

	for name, col := range r.Columns {
		if !isValidIdent(name) {
			return fmt.Errorf("resource %s: invalid column name %q", r.Name, name)
		}
		switch col.Type {
		case TypeString, TypeInteger, TypeNumber, TypeBoolean:
		default:
			return fmt.Errorf("resource %s: column %s has invalid type %q", r.Name, name, col.Type)
		}
	}

	r.known = make(map[string]struct{}, len(r.Columns)+len(r.SystemColumns)+len(builtinSystemColumns))
	for name := range r.Columns {
		r.known[name] = struct{}{}
	}
	for _, name := range builtinSystemColumns {
		r.known[name] = struct{}{}
	}
	for _, name := range r.SystemColumns {
		if !isValidIdent(name) {
			return fmt.Errorf("resource %s: invalid system column %q", r.Name, name)
		}
		r.known[name] = struct{}{}
	}

	for _, s := range r.Searchable {
		if _, ok := r.known[s]; !ok {
			return fmt.Errorf("resource %s: searchable column %q is not a declared or system column", r.Name, s)
		}
	}
	for _, f := range r.Filterable {
		if _, ok := r.known[f]; !ok {
			return fmt.Errorf("resource %s: filterable column %q is not a declared or system column", r.Name, f)
		}
	}
	for _, s := range r.DefaultSort {
		col := strings.TrimPrefix(s, "-")
		if _, ok := r.known[col]; !ok {
			return fmt.Errorf("resource %s: default_sort column %q is not a declared or system column", r.Name, col)
		}
	}

	r.readSet = toRoleSet(r.ReadRoles)
	r.writeSet = toRoleSet(r.WriteRoles)
	return nil
}

func (r *Resource) CanRead(role domain.UserRole) bool {
	if len(r.readSet) == 0 {
		return true
	}
	_, ok := r.readSet[role]
	return ok
}

func (r *Resource) CanWrite(role domain.UserRole) bool {
	if len(r.writeSet) == 0 {
		return false
	}
	_, ok := r.writeSet[role]
	return ok
}

func (r *Resource) IsSearchable(col string) bool {
	return contains(r.Searchable, col)
}

func (r *Resource) IsFilterable(col string) bool {
	return contains(r.Filterable, col)
}

func (r *Resource) IsKnownColumn(col string) bool {
	_, ok := r.known[col]
	return ok
}

func toRoleSet(roles []string) map[domain.UserRole]struct{} {
	set := make(map[domain.UserRole]struct{}, len(roles))
	for _, r := range roles {
		set[domain.UserRole(strings.TrimSpace(r))] = struct{}{}
	}
	return set
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func isValidIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '_':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
