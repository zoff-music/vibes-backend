# Opentracing placeholder

This package pretends to be opentracing, but is actually a wrapper around opentelemetry. 

The [opentracing](https://github.com/opentracing/specification/issues/163) and [jaeger](https://github.com/jaegertracing/jaeger-client-go) tracing packages have both been deprecated since 2022. In order to keep up with tracing standards and maintain compatibility with the grafana + tempo stack, we're switching to [opentelemetry](https://opentelemetry.io/).
This package allows projects to transition from these deprecated tracing APIs to Opentelemetry, without having to change a lot of code.

The following factors have been considered:

Pros:
- Allows us to switch from jaeger + opentracing to pure opentelemetry with a minimal amount of changes to the codebase

Cons:
- Adds an inevitable performance overhead
- Adds several layers of complexity, and hides the opentelemetry API
- Will cause new repos to also use the old opentracing interface
