import { type ClassValue, cn as cnMerge } from "cn";

export function cn(...inputs: ClassValue[]) {
  return cnMerge(inputs);
}