import type { ReactNode } from "react";
import type { NavId } from "@/lib/nav";
import { PAGE_TITLES } from "@/lib/nav";

type PageHeaderProps = {
  section: NavId;
  action?: ReactNode;
};

export function PageHeader({ section, action }: PageHeaderProps) {
  const { title, subtitle } = PAGE_TITLES[section];
  return (
    <div className="mb-8 flex items-end justify-between gap-4">
      <div>
        <h1 className="text-display">{title}</h1>
        <p className="mt-1 text-lg font-light leading-6 tracking-tight text-muted-foreground">
          {subtitle}
        </p>
      </div>
      {action}
    </div>
  );
}
