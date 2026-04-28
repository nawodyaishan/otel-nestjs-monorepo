import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { BatchSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { getWebAutoInstrumentations } from '@opentelemetry/auto-instrumentations-web';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import { resourceFromAttributes } from '@opentelemetry/resources';

const exporter = new OTLPTraceExporter({
  url: 'http://localhost:4318/v1/traces',
});

// In sdk-trace-web v2.x, span processors are configured via constructor options.
const provider = new WebTracerProvider({
  resource: resourceFromAttributes({
    'service.name': 'vite-react-web',
  }),
  spanProcessors: [new BatchSpanProcessor(exporter)],
});

provider.register({
  contextManager: new ZoneContextManager(),
});

registerInstrumentations({
  instrumentations: [
    getWebAutoInstrumentations({
      '@opentelemetry/instrumentation-fetch': {
        propagateTraceHeaderCorsUrls: [
          /http:\/\/localhost:3000.*/,
        ],
      },
    }),
  ],
});
