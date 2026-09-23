package controllers

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestParseControllers(t *testing.T) {
	known := []string{"Kafka", "KafkaTopic", "ServiceUser"}

	cases := []struct {
		name    string
		give    string
		want    []string
		wantErr string
	}{
		{name: "empty means all", give: "", want: known},
		{name: "star means all", give: "*", want: known},
		{name: "allow list", give: "Kafka,KafkaTopic", want: []string{"Kafka", "KafkaTopic"}},
		{name: "deny list", give: "*,-ServiceUser", want: []string{"Kafka", "KafkaTopic"}},
		{name: "case-insensitive with canonical result", give: "kafka,SERVICEUSER", want: []string{"Kafka", "ServiceUser"}},
		{name: "whitespace, duplicates and trailing comma", give: " Kafka , Kafka ,", want: []string{"Kafka"}},
		{name: "unknown kind", give: "Kafka,Redis", wantErr: `unknown kind "Redis", valid kinds are Kafka, KafkaTopic, ServiceUser`},
		{name: "unknown excluded kind", give: "*,-Redis", wantErr: `unknown kind "Redis"`},
		{name: "exclusion without star", give: "-Kafka", wantErr: `exclusions need "*" first`},
		{name: "allow mixed with exclusion", give: "Kafka,-KafkaTopic", wantErr: "not both"},
		{name: "allow mixed with star", give: "*,Kafka", wantErr: "not both"},
		{name: "everything excluded", give: "*,-Kafka,-KafkaTopic,-ServiceUser", wantErr: "nothing left to reconcile"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseControllers(tc.give, known)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Contains(t, err.Error(), fmt.Sprintf("%q", tc.give))
				return
			}

			require.NoError(t, err)
			want := make(kindSet, len(tc.want))
			for _, k := range tc.want {
				want[k] = true
			}
			require.Equal(t, want, got)
		})
	}
}

func TestValidateControllers(t *testing.T) {
	require.NoError(t, ValidateControllers(""))
	require.NoError(t, ValidateControllers("*,-Flink"))
	require.NoError(t, ValidateControllers("postgresql,Database"))

	err := ValidateControllers("Redis")
	require.ErrorContains(t, err, `unknown kind "Redis"`)
	// Every builder key is offered as a valid spelling, in sorted order.
	require.ErrorContains(t, err, "Clickhouse, ClickhouseDatabase")
	for k := range builders {
		require.Contains(t, err.Error(), k)
	}
}

func TestKindSetObjectsAndLists(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	typesOf := func(objs ...any) []reflect.Type {
		res := make([]reflect.Type, 0, len(objs))
		for _, o := range objs {
			res = append(res, reflect.TypeOf(o))
		}
		return res
	}

	t.Run("the full set covers every builder", func(t *testing.T) {
		all, err := parseControllers("*", knownKinds())
		require.NoError(t, err)
		require.Len(t, all.objects(scheme), len(builders))
		require.Len(t, all.lists(scheme), len(builders))
	})

	t.Run("a subset keeps only its kinds and their lists, sorted", func(t *testing.T) {
		s := kindSet{"KafkaTopic": true, "Kafka": true}
		require.Equal(t, typesOf(&v1alpha1.Kafka{}, &v1alpha1.KafkaTopic{}), typesOf(sliceToAny(s.objects(scheme))...))
		require.Equal(t, typesOf(&v1alpha1.KafkaList{}, &v1alpha1.KafkaTopicList{}), typesOf(sliceToAny(s.lists(scheme))...))
	})

	t.Run("an unknown kind yields nothing", func(t *testing.T) {
		s := kindSet{"Redis": true}
		require.Empty(t, s.objects(scheme))
		require.Empty(t, s.lists(scheme))
	})
}

func TestKindSetSecretSources(t *testing.T) {
	t.Parallel()

	all, err := parseControllers("*", knownKinds())
	require.NoError(t, err)
	require.Equal(t, allSecretSourceKinds, all.secretSources())

	require.Len(t, kindSet{"ServiceUser": true, "Kafka": true}.secretSources(), 1)
	require.Equal(t, "ServiceUser", kindSet{"ServiceUser": true, "Kafka": true}.secretSources()[0].kind)
	require.Empty(t, kindSet{"Kafka": true}.secretSources())
}

func sliceToAny[T any](in []T) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}
