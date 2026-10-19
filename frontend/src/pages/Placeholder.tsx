interface PlaceholderProps {
  title: string;
}

export function Placeholder({ title }: PlaceholderProps) {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <div className="text-center">
        <h1 className="text-lg font-semibold capitalize text-slate-900">{title}</h1>
        <p className="mt-1 text-sm text-slate-500">This section is coming soon.</p>
      </div>
    </div>
  );
}
