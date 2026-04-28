import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { LoggerModule } from 'nestjs-pino';
import { trace } from '@opentelemetry/api';

@Module({
  imports: [
    LoggerModule.forRoot({
      pinoHttp: {
        level: process.env.NODE_ENV !== 'production' ? 'debug' : 'info',
        customProps: () => {
          const activeSpan = trace.getActiveSpan();
          if (!activeSpan) return {};
          const spanContext = activeSpan.spanContext();
          return {
            trace_id: spanContext.traceId,
            span_id: spanContext.spanId,
          };
        },
      },
    }),
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
