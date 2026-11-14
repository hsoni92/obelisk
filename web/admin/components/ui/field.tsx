import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";

export function Field({
  label,
  className,
  ...inputProps
}: React.ComponentProps<typeof Input> & { label: string }) {
  return (
    <div className={cn("block space-y-1", className)}>
      <Label htmlFor={inputProps.id}>{label}</Label>
      <Input {...inputProps} />
    </div>
  );
}
