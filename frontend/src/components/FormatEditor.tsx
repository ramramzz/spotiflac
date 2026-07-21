import { useRef, useState, type ReactNode } from "react";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { RefreshCw } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { parseTemplate, SAMPLE_TEMPLATE_DATA, type TemplateData, type TemplateToken } from "@/lib/settings";
import { toastWithSound as toast } from "@/lib/toast-with-sound";
export interface FormatField {
    title: string;
    value: string;
    defaultValue: string;
    suffix?: string;
    placeholder?: string;
    column?: "left" | "right";
    titleAccessory?: ReactNode;
    onChange: (next: string) => void;
}
export function FormatEditor({ title, value, defaultValue, tokens, suffix, placeholder, sampleData = SAMPLE_TEMPLATE_DATA, onChange, fields, sideTokens = false }: {
    title?: string;
    value?: string;
    defaultValue?: string;
    tokens: TemplateToken[];
    suffix?: string;
    placeholder?: string;
    sampleData?: TemplateData;
    onChange?: (next: string) => void;
    fields?: FormatField[];
    sideTokens?: boolean;
}) {
    const resolvedFields: FormatField[] = fields ?? [{
            title: title ?? "",
            value: value ?? "",
            defaultValue: defaultValue ?? "",
            suffix,
            placeholder,
            onChange: onChange ?? (() => undefined),
        }];
    const inputRefs = useRef<Array<HTMLInputElement | null>>([]);
    const [activeIndex, setActiveIndex] = useState<number | null>(null);
    const insertToken = (token: string) => {
        if (activeIndex === null) {
            return;
        }
        const idx = activeIndex < resolvedFields.length ? activeIndex : 0;
        const field = resolvedFields[idx];
        const input = inputRefs.current[idx];
        const current = field.value ?? "";
        let next: string;
        let caret: number;
        if (input && input.selectionStart !== null && input.selectionEnd !== null) {
            const start = input.selectionStart;
            const end = input.selectionEnd;
            next = current.slice(0, start) + token + current.slice(end);
            caret = start + token.length;
        }
        else {
            next = current + token;
            caret = next.length;
        }
        field.onChange(next);
        void navigator.clipboard?.writeText(token).catch(() => undefined);
        toast.success(`${token} copied`);
        requestAnimationFrame(() => {
            if (input) {
                input.focus();
                input.setSelectionRange(caret, caret);
            }
        });
    };
    const renderField = (field: FormatField, idx: number) => {
        const preview = parseTemplate(field.value, sampleData) + (field.suffix ?? "");
        return (<div key={idx} className="space-y-3">
            {(field.title || field.titleAccessory) && (<div className="flex items-center justify-between gap-3 min-h-6">
              {field.title ? <Label className="text-sm font-semibold">{field.title}</Label> : <span />}
              {field.titleAccessory}
            </div>)}
            <div className="relative">
              <Input ref={(el) => { inputRefs.current[idx] = el; }} value={field.value} placeholder={field.placeholder} onFocus={() => setActiveIndex(idx)} onChange={(e) => field.onChange(e.target.value)} className="font-mono text-sm pr-9"/>
              <button type="button" onClick={() => field.onChange(field.defaultValue)} className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors">
                <RefreshCw className="h-4 w-4"/>
              </button>
            </div>
            <div className="rounded-lg border bg-muted/40 px-3 py-2">
              <div className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">Preview</div>
              <div className="font-mono text-sm break-all">{preview || <span className="text-muted-foreground italic">empty</span>}</div>
            </div>
          </div>);
    };
    const indexed = resolvedFields.map((field, idx) => ({ field, idx }));
    const hasColumns = indexed.some(({ field }) => field.column === "right");
    const tokenList = (<div className="flex flex-wrap gap-2">
          {tokens.map((token) => {
            const disabled = activeIndex === null;
            return (<Tooltip key={token.key}>
              <TooltipTrigger asChild>
                <button type="button" disabled={disabled} onMouseDown={(e) => e.preventDefault()} onClick={() => insertToken(token.key)} className="rounded-md border bg-background px-2.5 py-1 text-xs font-mono text-muted-foreground transition-colors enabled:hover:text-foreground enabled:hover:border-primary/50 disabled:opacity-40 disabled:cursor-not-allowed">
                  {token.key}
                </button>
              </TooltipTrigger>
              <TooltipContent side="top">
                <span className="font-mono">{disabled ? "Click a field first" : (token.example || "—")}</span>
              </TooltipContent>
            </Tooltip>);
        })}
        </div>);
    if (sideTokens && !hasColumns) {
        const hasTitleRow = indexed.some(({ field }) => field.title || field.titleAccessory);
        return (<div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-start">
            <div className="space-y-4 min-w-0 md:pr-6 md:border-r border-border">
              {indexed.map(({ field, idx }) => renderField(field, idx))}
            </div>
            <div className="min-w-0">
              {hasTitleRow && <div aria-hidden className="hidden md:block min-h-6 mb-3"/>}
              {tokenList}
            </div>
          </div>);
    }
    return (<div className="space-y-6">
        {hasColumns ? (<div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-stretch">
            <div className="space-y-4 min-w-0 md:pr-6 md:border-r border-border">
              {indexed.filter(({ field }) => field.column !== "right").map(({ field, idx }) => renderField(field, idx))}
            </div>
            <div className="space-y-4 min-w-0">
              {indexed.filter(({ field }) => field.column === "right").map(({ field, idx }) => renderField(field, idx))}
            </div>
          </div>) : (<div className="space-y-4">
            {indexed.map(({ field, idx }) => renderField(field, idx))}
          </div>)}
        {tokenList}
      </div>);
}
