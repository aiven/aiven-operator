---
title: "Resource generation"
---

# Resource generation

Aiven Kubernetes Operator generates service configs code (also known as _user configs_)
and documentation
from public [service types schema][service-types].

## The flow overview

When a new schema is issued on the API,
a cron job fetches it, parses, patches, and saves in a shared library — [tools][tools].

When the library is updated, 
the GitHub [dependabot](https://github.com/dependabot) creates a PR with new `tools` version to the dependant repositories, 
like Aiven Kubernetes Operator and Aiven Terraform Provider.

Then the [make generate](#make-generate) command is called by GitHub Actions.
And the PR is ready for review.

```mermaid
flowchart TB
    API(Aiven API) <-.->|polls schema updates| Tools([tools repository])
    Bot(dependabot) <-.->|polls updates| Tools 
    Bot-->|new tools version|UpdateOP[/"✨ $ make generate ✨"/]
    UpdateOP-->|pull request| OP([operator repository])
```

## make generate

The command runs several generators in a certain sequence.
First, the user config generator is called.
Then [controller-gen][controller-gen] cli.
Then [API reference][api-reference] docs generator.  
Here how it goes in the details:

1. User config generator creates Go structs (k8s api compatible objects) with docstrings, 
   validation rules and constraints (immutable, maxLength, etc)
2. [controller-gen][controller-gen] implements k8s [Object Interface][object-interface],
   generates [CRDs][crd] for those objects, 
   creates charts for cluster roles and webhooks. 
3. Docs generator creates [API reference][api-reference] out of CRDs:
    1. it looks for an example file for the given CRD kind in `./<api-reference-docs>/example/`,
       if it finds one, it validates that with the CRD. 
       Each CRD has an OpenAPI v3 schema as a part of it. 
       This is also used by Kubernetes itself to validate user input.
    2. generates full spec reference out of the schema
    3. creates a markdown file with spec and example (if exists)
4. _todo:_ Releaser updates CRDs, webhooks, cluster roles charts, and the changelog. 
   Prepares a new release.  

[tools]: https://github.com/aiven/aiven-go-client/tree/master/tools/exp
[service-types]: https://api.aiven.io/doc/#tag/Service/operation/ListPublicServiceTypes
[api-reference]: ../api-reference/index.md
[controller-gen]: https://book.kubebuilder.io/reference/controller-gen.html
[object-interface]: https://github.com/kubernetes/apimachinery/blob/76eb944e266d86623b855a30ff51c4a371568da6/pkg/runtime/interfaces.go#L323
[crd]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/

```mermaid
flowchart TB
    Make[/$ make generate/]-->Generator(userconfig generator<br> creates/updates structs using updated spec)
    Generator-->|go: KafkaUserConfig struct| K8S(controller-gen<br> implements k8s object interface<br> for structs)
    K8S-->|go files| CRD(controller-gen<br> creates CRDs out of structs)
    CRD-->|CRD: aiven.io_kafkas.yaml| Docs(docs generator)
    subgraph API reference generation
        Docs-->|aiven.io_kafkas.yaml|Reference(creates reference<br> out of CRD)
        Docs-->|examples/kafka.yaml,<br> aiven.io_kafkas.yaml|Examples(validates example<br> using CRD)
        Examples--> Markdown(creates docs out of CRDs, adds examples)
        Reference-->Markdown(kafka.md)
    end
    CRD-.->|yaml files|Releaser(<i>todo:</i> releaser<br> updates helm charts)
    Releaser-.->NewVersion("New version 🎉")
    Markdown-->NewVersion
    
```
