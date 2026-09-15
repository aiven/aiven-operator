package controllers

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// kindSet is the set of kinds selected by --controllers flag.
type kindSet map[string]bool

// errRefKindDisabled means a resource references a kind that has no controller to make it ready.
var errRefKindDisabled = errors.New("referenced kind is disabled by --controllers")

// ValidateControllers reports whether spec is an acceptable value for the --controllers flag.
func ValidateControllers(spec string) error {
	_, err := parseControllers(spec, knownKinds())
	return err
}

// knownKinds lists the kinds accepted by --controllers, sorted for stable logs and errors.
func knownKinds() []string {
	kinds := make([]string, 0, len(builders))
	for k := range builders {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return kinds
}

// parseControllers resolves a --controllers value against known kinds. Names match
// case-insensitively; the result uses the canonical spelling.
func parseControllers(spec string, known []string) (kindSet, error) {
	canonical := make(map[string]string, len(known))
	for _, k := range known {
		canonical[strings.ToLower(k)] = k
	}
	resolve := func(name string) (string, error) {
		if k, ok := canonical[strings.ToLower(name)]; ok {
			return k, nil
		}
		return "", fmt.Errorf("controllers %q: unknown kind %q, valid kinds are %s", spec, name, strings.Join(known, ", "))
	}

	all := false
	var allow, deny []string
	for _, tok := range strings.Split(spec, ",") {
		tok = strings.TrimSpace(tok)
		switch {
		case tok == "":
		case tok == "*":
			all = true
		case strings.HasPrefix(tok, "-"):
			deny = append(deny, strings.TrimPrefix(tok, "-"))
		default:
			allow = append(allow, tok)
		}
	}

	if len(allow) > 0 && (all || len(deny) > 0) {
		return nil, fmt.Errorf(`controllers %q: use either an allow list "Kind,Kind" or exclusions "*,-Kind", not both`, spec)
	}
	if len(deny) > 0 && !all {
		return nil, fmt.Errorf(`controllers %q: exclusions need "*" first, as in "*,-Kind"`, spec)
	}

	enabled := make(kindSet, len(known))
	if all || len(allow) == 0 {
		for _, k := range known {
			enabled[k] = true
		}
	}
	for _, name := range allow {
		k, err := resolve(name)
		if err != nil {
			return nil, err
		}
		enabled[k] = true
	}
	for _, name := range deny {
		k, err := resolve(name)
		if err != nil {
			return nil, err
		}
		delete(enabled, k)
	}
	if len(enabled) == 0 {
		return nil, fmt.Errorf("controllers %q: every kind is excluded, nothing left to reconcile", spec)
	}
	return enabled, nil
}

func (s kindSet) has(kind string) bool {
	return s[kind]
}

// checkRef rejects a reference to a disabled kind. A nil set has no restrictions.
func (s kindSet) checkRef(gvk schema.GroupVersionKind) error {
	if s == nil || s.has(gvk.Kind) {
		return nil
	}
	return fmt.Errorf("%w: enable %s or do not reference it", errRefKindDisabled, gvk.Kind)
}

// objects returns one instance of every enabled kind that is an AivenManagedObject, in kind order.
func (s kindSet) objects(scheme *runtime.Scheme) []v1alpha1.AivenManagedObject {
	res := make([]v1alpha1.AivenManagedObject, 0, len(s))
	types, kinds := sortedKinds(scheme)
	for _, kind := range kinds {
		if !s.has(kind) {
			continue
		}
		if obj, ok := reflect.New(types[kind]).Interface().(v1alpha1.AivenManagedObject); ok {
			res = append(res, obj)
		}
	}
	return res
}

// lists returns the list type of every enabled kind, in kind order.
func (s kindSet) lists(scheme *runtime.Scheme) []client.ObjectList {
	res := make([]client.ObjectList, 0, len(s))
	types, kinds := sortedKinds(scheme)
	for _, kind := range kinds {
		if !s.has(strings.TrimSuffix(kind, "List")) {
			continue
		}
		if list, ok := reflect.New(types[kind]).Interface().(client.ObjectList); ok {
			res = append(res, list)
		}
	}
	return res
}

// secretSources returns the enabled kinds that support connInfoSecretSource.
func (s kindSet) secretSources() []secretSourceKind {
	res := make([]secretSourceKind, 0, len(allSecretSourceKinds))
	for _, k := range allSecretSourceKinds {
		if s.has(k.kind) {
			res = append(res, k)
		}
	}
	return res
}

// sortedKinds returns the scheme's types for the aiven.io group version and their kinds in sorted order.
func sortedKinds(scheme *runtime.Scheme) (map[string]reflect.Type, []string) {
	types := scheme.KnownTypes(v1alpha1.GroupVersion)
	kinds := make([]string, 0, len(types))
	for k := range types {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return types, kinds
}
