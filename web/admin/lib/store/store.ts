import { configureStore } from "@reduxjs/toolkit";
import { filtersReducer } from "./filters-slice";
import { obeliskApi } from "./obelisk-api";

export function makeStore() {
  return configureStore({
    reducer: {
      filters: filtersReducer,
      [obeliskApi.reducerPath]: obeliskApi.reducer,
    },
    middleware: (getDefaultMiddleware) =>
      getDefaultMiddleware().concat(obeliskApi.middleware),
    devTools: process.env.NODE_ENV !== "production",
  });
}

export type AppStore = ReturnType<typeof makeStore>;
export type RootState = ReturnType<AppStore["getState"]>;
export type AppDispatch = AppStore["dispatch"];