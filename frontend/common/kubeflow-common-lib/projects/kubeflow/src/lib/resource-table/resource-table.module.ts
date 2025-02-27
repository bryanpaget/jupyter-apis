import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ResourceTableComponent } from './resource-table.component';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDividerModule } from '@angular/material/divider';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatTableModule } from '@angular/material/table';
import { MatPaginatorModule, MatPaginatorIntl } from '@angular/material/paginator';
import { PaginatorIntlComponent } from './paginator/paginator.component';
import { StatusComponent } from './status/status.component';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { ActionComponent } from './action/action.component';
import { MatChipsModule } from '@angular/material/chips';
import { FontAwesomeModule } from '@fortawesome/angular-fontawesome';
import { ActionButtonComponent } from './action-button/action-button.component';
import { IconModule } from '../icon/icon.module';
import { TableComponent } from './table/table.component';
import { DateTimeModule } from '../date-time/date-time.module';
import { PopoverModule } from '../popover/popover.module';
import { TableChipsListComponent } from './chips-list/chips-list.component';
import { ComponentValueComponent } from './component-value/component-value.component';
import { PortalModule } from '@angular/cdk/portal';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

@NgModule({
  imports: [
    CommonModule,
    BrowserAnimationsModule,
    MatTableModule,
    MatTooltipModule,
    MatProgressSpinnerModule,
    MatDividerModule,
    MatCardModule,
    MatButtonModule,
    MatChipsModule,
    MatMenuModule,
    MatPaginatorModule,
    PortalModule,
    FontAwesomeModule,
    MatIconModule,
    IconModule,
    DateTimeModule,
    PopoverModule,
  ],
  declarations: [
    ResourceTableComponent,
    StatusComponent,
    ActionComponent,
    ActionButtonComponent,
    TableChipsListComponent,
    TableComponent,
    ComponentValueComponent,
  ],
  exports: [ResourceTableComponent, TableComponent],
  providers: [{provide: MatPaginatorIntl, useClass: PaginatorIntlComponent}],
})
export class ResourceTableModule {}
