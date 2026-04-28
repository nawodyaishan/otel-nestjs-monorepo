import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { LoggerModule } from 'nestjs-pino';

@Module({
  imports: [
    LoggerModule.forRoot({
      pinoHttp: {
        level: process.env.NODE_ENV !== 'production' ? 'debug' : 'info',
        // trace_id / span_id are injected automatically by @opentelemetry/instrumentation-pino
        // (included in getNodeAutoInstrumentations) which patches Pino before NestJS bootstraps.
      },
    }),
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
