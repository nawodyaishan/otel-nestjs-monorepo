import { Controller, Get } from '@nestjs/common';
import { AppService } from './app.service';
import { PinoLogger } from 'nestjs-pino';
import { trace } from '@opentelemetry/api';

@Controller()
export class AppController {
  constructor(
    private readonly appService: AppService,
    private readonly logger: PinoLogger
  ) {
    this.logger.setContext(AppController.name);
  }

  @Get('hello')
  getHello(): string {
    const activeSpan = trace.getActiveSpan();
    if (activeSpan) {
      activeSpan.setAttribute('custom.hello.attribute', 'Greeting executed');
      activeSpan.addEvent('Executing getHello handler');
    }
    
    this.logger.info('Handling GET /hello request - testing correlated JSON logging');
    return this.appService.getHello();
  }
}
