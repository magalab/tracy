type TokenFormProps = {
  value: string;
  onChange: (value: string) => void;
  onSave: () => void;
};

export function TokenForm({ value, onChange, onSave }: TokenFormProps) {
  return (
    <div className="token-form">
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        type="password"
        placeholder="API key"
        onKeyDown={(event) => event.key === "Enter" && onSave()}
      />
      <button onClick={onSave}>Connect</button>
    </div>
  );
}
