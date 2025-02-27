import {
  PropertyValue,
  StatusValue,
  ActionListValue,
  ActionIconValue,
  ActionButtonValue,
  MenuValue,
  DialogConfig,
  ComponentValue,
  TableConfig,
  TABLE_THEME,
  DateTimeValue,
} from 'kubeflow';
import { ServerTypeComponent } from './server-type/server-type.component';
import { ProtBComponent } from './protb-icon/protb-icon.component';

// --- Configs for the Confirm Dialogs ---
export function getDeleteDialogConfig(name: string): DialogConfig {
  return {
    title: $localize`Are you sure you want to delete this notebook server? ${name}`,
    message: $localize`Warning: Your data might be lost if the notebook server
                       is not backed by persistent storage`,
    accept: $localize`DELETE`,
    confirmColor: 'warn',
    cancel: $localize`CANCEL`,
    error: '',
    applying: $localize`DELETING`,
    width: '600px',
  };
}

export function getStopDialogConfig(name: string): DialogConfig {
  return {
    title: $localize`Are you sure you want to stop this notebook server? ${name}`,
    message: $localize`Warning: Your data might be lost if the notebook server
                       is not backed by persistent storage.`,
    accept: $localize`STOP`,
    confirmColor: 'primary',
    cancel: $localize`CANCEL`,
    error: '',
    applying: $localize`STOPPING`,
    width: '600px',
  };
}

// --- Config for the Resource Table ---
export const defaultConfig: TableConfig = {
  columns: [
    {
      matHeaderCellDef: $localize`Status`,
      matColumnDef: 'status',
      value: new StatusValue(),
    },
    {
      matHeaderCellDef: $localize`Name`,
      matColumnDef: 'name',
      style: { width: '25%' },
      value: new PropertyValue({
        field: 'name',
        tooltipField: 'name',
        truncate: true,
      }),
    },
    {
      matHeaderCellDef: '',
      matColumnDef: 'prot-b',
      value: new ComponentValue({
        component: ProtBComponent,
      }),
    },
    {
      matHeaderCellDef: $localize`Type`,
      matColumnDef: 'type',
      value: new ComponentValue({
        component: ServerTypeComponent,
      }),
    },
    {
      matHeaderCellDef: $localize`Age`,
      matColumnDef: 'age',
      style: { width: '12%' },
      textAlignment: 'right',
      value: new PropertyValue({ field: 'age', truncate: true }),
    },
    {
      matHeaderCellDef: $localize`Last activity`,
      matColumnDef: 'last_activity',
      value: new DateTimeValue({ field: 'last_activity' }),
    },
    {
      matHeaderCellDef: $localize`Image`,
      matColumnDef: 'image',
      style: { width: '30%' },
      value: new PropertyValue({
        field: 'shortImage',
        popoverField: 'image',
        truncate: true,
        style: { maxWidth: '300px' },
      }),
    },
    {
      matHeaderCellDef: $localize`GPUs`,
      matColumnDef: 'gpus',
      style: { width: '8%' },
      textAlignment: 'right',
      value: new PropertyValue({
        field: 'gpus.count',
        tooltipField: 'gpus.message',
      }),
    },
    {
      matHeaderCellDef: $localize`CPUs`,
      matColumnDef: 'cpu',
      style: { width: '8%' },
      textAlignment: 'right',
      value: new PropertyValue({ field: 'cpu' }),
    },
    {
      matHeaderCellDef: $localize`Memory`,
      matColumnDef: 'memory',
      style: { width: '8%' },
      textAlignment: 'right',
      value: new PropertyValue({ field: 'memory' }),
    },
    {
      matHeaderCellDef: $localize`Volumes`,
      matColumnDef: 'volumes',
      value: new MenuValue({ field: 'volumes', itemsIcon: 'storage' }),
    },

    {
      matHeaderCellDef: '',
      matColumnDef: 'actions',
      value: new ActionListValue([
        new ActionButtonValue({
          name: 'connect',
          tooltip: $localize`Connect to this notebook server`,
          color: 'primary',
          field: 'connectAction',
          text: $localize`CONNECT`,
        }),
        new ActionIconValue({
          name: 'start-stop',
          tooltipInit: $localize`Stop this notebook server`,
          tooltipReady: $localize`Start this notebook server`,
          color: '',
          field: 'startStopAction',
          iconInit: 'material:stop',
          iconReady: 'material:play_arrow',
        }),
        new ActionIconValue({
          name: 'delete',
          tooltip: $localize`Delete this notebook server`,
          color: '',
          field: 'deleteAction',
          iconReady: 'material:delete',
        }),
      ]),
    },
  ],
};

export const defaultVolumeConfig = {
  columns: [
    {
      matHeaderCellDef: $localize`Status`,
      matColumnDef: 'status',
      value: new StatusValue(),
    },
    {
      matHeaderCellDef: $localize`Name`,
      matColumnDef: 'name',
      style: { width: '25%' },
      value: new PropertyValue({
        field: 'name',
        tooltipField: 'name',
        truncate: true,
      }),
    },
    {
      matHeaderCellDef: '',
      matColumnDef: 'prot-b',
      value: new ComponentValue({
        component: ProtBComponent,
      }),
    },
    {
      matHeaderCellDef: $localize`Size`,
      matColumnDef: 'size',
      value: new PropertyValue({ field: 'size' }),
    },
    {
      matHeaderCellDef: $localize`Used By`,
      matColumnDef: 'usedBy',
      value: new PropertyValue({ field: 'usedBy' }),
    },
    {
      matHeaderCellDef: '',
      matColumnDef: 'actions',
      value: new ActionListValue([
        new ActionIconValue({
          name: 'delete',
          tooltip: $localize`Delete Volume`,
          color: 'warn',
          field: 'deleteAction',
          iconReady: 'material:delete',
        }),
      ]),
  },
  ],
};

export function getDeleteVolumeDialogConfig(name: string): DialogConfig {
  return { // TODO key + param
    title: $localize`Delete Volume '${name}'`,
    message: $localize`Warning: All data in this volume will be lost.`,
    accept: $localize`DELETE`,
    confirmColor: 'warn',
    cancel: $localize`CANCEL`,
    error: '',
    applying: $localize`DELETING`,
    width: '600px',
  };
}

// --- Config for the Cost Table ---
export const defaultCostConfig = {
  columns: [
    {
      matHeaderCellDef: $localize`CPUs`,
      matColumnDef: 'cpus',
      value: new PropertyValue({ field: 'cpuCost' }),
    },
    {
      matHeaderCellDef: $localize`GPUs`,
      matColumnDef: 'gpus',
      value: new PropertyValue({ field: 'gpuCost' }),
    },
    {
      matHeaderCellDef: $localize`RAM`,
      matColumnDef: 'ram',
      value: new PropertyValue({ field: 'ramCost' }),
    },
    {
      matHeaderCellDef: $localize`Storage`,
      matColumnDef: 'storage',
      value: new PropertyValue({ field: 'pvCost' }),
    },
    {
      matHeaderCellDef: $localize`Total`,
      matColumnDef: 'total',
      value: new PropertyValue({ field: 'totalCost' }),
    },
  ],
};
