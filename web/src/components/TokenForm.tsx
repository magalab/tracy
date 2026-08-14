type TokenFormProps = {
  value: string;
  connected: boolean;
  onChange: (value: string) => void;
  onSave: () => void;
};

export function TokenForm({ value, connected, onChange, onSave }: TokenFormProps) {
  return (
    <div className={`token-form ${connected ? "connected" : ""}`}>
      <div className="token-state">
        <span className="token-state-dot" />
        <span>
          <b>{connected ? "Connected" : "Connect project"}</b>
          <small>{connected ? "API key active" : "Enter project API key"}</small>
        </span>
      </div>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        type="password"
        aria-label="Project API key"
        placeholder="Paste API key"
        onKeyDown={(event) => event.key === "Enter" && onSave()}
      />
      <button onClick={onSave}>{connected ? "Reconnect" : "Connect"}</button>
    </div>
  );
}
