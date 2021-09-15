{{ define "packages" }}

---
title: "API Reference"
linkTitle: "API Reference"
weight: 90
---

{{ range .packages }}
    ## {{ packageDisplayName . }}

    {{ with (index .GoPackages 0 )}}
        {{ with .DocComments }}
            {{ safe (renderComments .) }}
        {{ end }}
    {{ end }}

    {{ range (visibleTypes (sortedTypes .Types))}}
        {{ template "type" .  }}
    {{ end }}
{{ end }}

## References

Generated with [gen-crd-api-reference-docs](https://github.com/ahmetb/gen-crd-api-reference-docs) {{ with .gitCommit }} on git commit `{{ . }}`{{end}}.

{{ end }}

{{ define "type" }}
  ### {{ .Name.Name }} {{ if eq .Kind "Alias" }}(`{{.Underlying}}` alias) {{ end }}
  {{ if ( .CommentLines ) }} {{ template "joinCommentLines" .CommentLines }}  {{ end }}
  {{ if ( .Members ) -}} {{ template "membersTable" . }} {{ end }}
{{ end }}


{{ define "appearsOn" }} 
  {{ with ( typeReferences . ) -}}
    {{- $prev := "" -}}
    {{- range . }}
      - {{ $prev = . }} [{{ typeDisplayName . }}]({{ linkForType . }})
    {{- end }}
  {{- end }}
{{- end }}


{{ define "joinCommentLines" -}}
  {{- $prev := "" -}}
  {{ range $index, $line := . }}
    {{- if not ( eq 0 ( len $line ) ) }}
      {{- if not ( eq ( index $line 0 ) '+'  ) }}
        {{- . -}} {{- if $prev }} {{ end -}}
        {{- $prev = . -}}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}


{{ define "membersTable" -}}
  | Field | Description|
  |---|---|
  {{- range .Members }}
    {{- if not ( hiddenMember . ) }}
      | {{- template "memberField" . -}} | {{- template "memberDescription" . -}} |
    {{- end }}
  {{- end }}
{{- end }}


{{ define "memberField" -}}
`{{ fieldName . -}}` <br> {{- template "memberFieldType" . -}}
{{- end }}


{{ define "memberFieldType" -}}
  {{- if linkForType .Type -}}
      [{{- typeDisplayName .Type -}}]({{- linkForType .Type -}}) 
  {{- else -}} 
    {{- typeDisplayName .Type -}} 
  {{- end -}} 
{{- end }}


{{ define "memberDescription" -}}
  {{- if fieldEmbedded . -}}
    (Members of `{{ fieldName . }}`are embedded into this type.)
  {{- else if isOptionalMember . -}}
    (Optional)
  {{- else if ( not ( eq 0 ( len ( renderComments .CommentLines ) ) ) ) -}}
    {{- template "joinCommentLines" .CommentLines -}}
  {{- else if ( eq ( .Type.Name.Name ) "ObjectMeta" ) -}}
    Refer to the Kubernetes API documentation for the fields of the `metadata` field.
  {{- else -}}
    N/A
  {{- end -}}
{{- end }}

