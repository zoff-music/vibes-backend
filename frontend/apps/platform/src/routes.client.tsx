import type { ComponentType } from 'react';
import { type RouteObject, redirect } from 'react-router';
import App, { type SSRInitialData } from './App';

type LazyModule = {
  default: ComponentType;
};

const lazyRoute = (loader: () => Promise<LazyModule>) => async () => {
  const module = await loader();
  return { Component: module.default };
};

export function createClientRoutes(
  initialData?: SSRInitialData,
): RouteObject[] {
  return [
    {
      path: '/',
      element: <App initialData={initialData} />,
      children: [
        {
          index: true,
          lazy: lazyRoute(() => import('./pages/Home')),
        },
        {
          path: 'rooms/create',
          lazy: lazyRoute(() => import('./pages/CreateRoom')),
        },
        {
          path: 'rooms/:id',
          lazy: lazyRoute(() => import('./pages/Room')),
        },
        {
          path: 'callback',
          lazy: lazyRoute(() => import('./pages/Callback')),
        },
        {
          path: 'admin',
          lazy: lazyRoute(() => import('./pages/Admin')),
        },
        {
          path: '*',
          loader: () => redirect('/'),
        },
      ],
    },
  ];
}
