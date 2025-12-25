import React, { useState } from 'react';
import { FormInput, FormOutput } from '@/types/schemas';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Loader2, AlertCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { OptionalComment, normalizeOptionalComment } from './OptionalComment';

interface Props {
  requestId: string;
  input: FormInput;
  onSubmit: (output: FormOutput) => Promise<void>;
  loading?: boolean;
}

function getSchemaString(v: any): string | undefined {
  return typeof v === 'string' && v.trim().length > 0 ? v : undefined;
}

function getFirstExample(v: any): string | undefined {
  if (!v) return undefined;
  if (Array.isArray(v) && v.length > 0) {
    const ex = v[0];
    if (ex === null || ex === undefined) return undefined;
    return String(ex);
  }
  return undefined;
}

export const FormDialog: React.FC<Props> = ({ input, onSubmit, loading }) => {
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [comment, setComment] = useState('');

  const schema = input.schema || { properties: {}, required: [] };
  const properties = schema.properties || {};
  const required = schema.required || [];

  const schemaTitle = getSchemaString(schema?.title);
  const schemaDescription = getSchemaString(schema?.description);

  const validateField = (name: string, value: any) => {
    const fieldSchema = properties[name];
    if (!fieldSchema) return null;

    if (required.includes(name) && (value === undefined || value === '' || value === null)) {
      return 'FIELD_REQUIRED';
    }

    if (fieldSchema.type === 'string') {
      if (fieldSchema.minLength && String(value).length < fieldSchema.minLength) {
        return `MIN_LENGTH_${fieldSchema.minLength}`;
      }
      if (fieldSchema.maxLength && String(value).length > fieldSchema.maxLength) {
        return `MAX_LENGTH_${fieldSchema.maxLength}`;
      }
      if (fieldSchema.pattern && !new RegExp(fieldSchema.pattern).test(value)) {
        return 'INVALID_FORMAT';
      }
      if (fieldSchema.format === 'email' && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
        return 'INVALID_EMAIL';
      }
    }

    if (fieldSchema.type === 'number') {
      const num = Number(value);
      if (isNaN(num)) return 'INVALID_NUMBER';
      if (fieldSchema.minimum !== undefined && num < fieldSchema.minimum) {
        return `MIN_VALUE_${fieldSchema.minimum}`;
      }
      if (fieldSchema.maximum !== undefined && num > fieldSchema.maximum) {
        return `MAX_VALUE_${fieldSchema.maximum}`;
      }
    }

    return null;
  };

  const handleChange = (name: string, value: any) => {
    setFormData(prev => ({ ...prev, [name]: value }));
    
    const error = validateField(name, value);
    setErrors(prev => {
      const newErrors = { ...prev };
      if (error) {
        newErrors[name] = error;
      } else {
        delete newErrors[name];
      }
      return newErrors;
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Validate all fields
    const newErrors: Record<string, string> = {};
    let isValid = true;

    Object.keys(properties).forEach(name => {
      const error = validateField(name, formData[name]);
      if (error) {
        newErrors[name] = error;
        isValid = false;
      }
    });

    setErrors(newErrors);

    if (!isValid) return;

    setSubmitting(true);
    const c = normalizeOptionalComment(comment);
    await onSubmit({ data: formData, ...(c ? { comment: c } : {}) });
    setSubmitting(false);
  };

  const renderField = (name: string, fieldSchema: any) => {
    const isRequired = required.includes(name);
    const error = errors[name];

    const labelText = getSchemaString(fieldSchema?.title) || name;
    const fieldDescription = getSchemaString(fieldSchema?.description);
    const showKey = labelText !== name;
    const placeholderHint =
      getFirstExample(fieldSchema?.examples) ||
      (fieldSchema?.default !== undefined && fieldSchema?.default !== null ? String(fieldSchema.default) : undefined);

    if (fieldSchema.type === 'boolean') {
      return (
        <div className="space-y-1 py-2">
          <div className="flex items-center space-x-2">
            <Checkbox
              id={name}
              checked={!!formData[name]}
              onCheckedChange={(checked) => handleChange(name, checked)}
              className="border-primary/50 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground rounded-none"
            />
            <Label
              htmlFor={name}
              className="text-sm font-mono cursor-pointer uppercase"
            >
              {labelText} {isRequired && <span className="text-destructive">*</span>}
            </Label>
          </div>
          {showKey && (
            <div className="text-[10px] font-mono text-muted-foreground/60">
              KEY: {name}
            </div>
          )}
          {fieldDescription && (
            <div className="text-xs font-mono text-muted-foreground whitespace-pre-wrap leading-relaxed">
              {fieldDescription}
            </div>
          )}
        </div>
      );
    }

    return (
      <div className="space-y-2">
        <Label 
          htmlFor={name}
          className="text-xs font-mono uppercase text-muted-foreground"
        >
          {labelText} {isRequired && <span className="text-destructive">*</span>}
        </Label>
        {showKey && (
          <div className="text-[10px] font-mono text-muted-foreground/60">
            KEY: {name}
          </div>
        )}
        {fieldDescription && (
          <div className="text-xs font-mono text-muted-foreground whitespace-pre-wrap leading-relaxed">
            {fieldDescription}
          </div>
        )}
        <Input
          id={name}
          type={fieldSchema.format === 'password' ? 'password' : fieldSchema.type === 'number' ? 'number' : 'text'}
          value={formData[name] ?? ''}
          onChange={(e) => handleChange(name, e.target.value)}
          className={cn(
            "cyber-input h-10 font-mono text-sm",
            error && "border-destructive/50 focus:ring-destructive"
          )}
          placeholder={placeholderHint ? placeholderHint : `ENTER_${name.toUpperCase()}...`}
        />
        {error && (
          <div className="flex items-center text-xs text-destructive font-mono mt-1">
            <AlertCircle className="h-3 w-3 mr-1" />
            {error}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="bg-background p-6 md:p-8 min-h-[400px] flex flex-col relative">
      <div className="space-y-4 mb-6">
        <h2 className="text-2xl font-display font-bold tracking-tight text-primary uppercase">
          {input.title}
        </h2>
        {schemaTitle && schemaTitle !== input.title && (
          <div className="text-sm font-mono text-muted-foreground whitespace-pre-wrap leading-relaxed">
            {schemaTitle}
          </div>
        )}
        {schemaDescription && (
          <div className="text-sm font-mono text-muted-foreground whitespace-pre-wrap leading-relaxed">
            {schemaDescription}
          </div>
        )}
        <div className="h-px w-full bg-border" />
      </div>

      <ScrollArea className="flex-1 -mx-2 px-2 mb-6 max-h-[60vh]">
        <form id="schema-form" onSubmit={handleSubmit} className="space-y-6 py-2">
          {Object.entries(properties).map(([name, fieldSchema]) => (
            <div key={name}>
              {renderField(name, fieldSchema)}
            </div>
          ))}
        </form>
      </ScrollArea>

      <div className="pt-4 border-t border-border space-y-3">
        <OptionalComment value={comment} onChange={setComment} disabled={loading || submitting} />

        <div className="flex justify-end">
          <Button
            type="submit"
            form="schema-form"
            className="cyber-button min-w-[140px]"
            disabled={loading || submitting}
          >
            {submitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            SUBMIT_FORM
          </Button>
        </div>
      </div>
    </div>
  );
};
