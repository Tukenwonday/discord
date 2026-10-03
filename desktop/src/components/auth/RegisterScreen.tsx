import { useState, type React } from "react";
import { Link } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useAuth } from "@/hooks/useAuth";

const USERNAME_PATTERN = /^[a-z0-9_.]+$/;

/** Account creation with the contract's username and password rules. */
export function RegisterScreen(): React.JSX.Element {
  const { register, error, isLoading } = useAuth();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function onSubmit(event: React.FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const errors: Record<string, string> = {};

    const normalized = username.trim().toLowerCase();
    if (normalized.length < 3 || normalized.length > 32) {
      errors.username = "Usernames are 3 to 32 characters";
    } else if (!USERNAME_PATTERN.test(normalized)) {
      errors.username = "Use lowercase letters, numbers, dots or underscores";
    }

    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      errors.email = "Enter a valid email address";
    }

    if (password.length < 8 || password.length > 128) {
      errors.password = "Passwords are 8 to 128 characters";
    } else if (!/[a-zA-Z]/.test(password) || !/\d/.test(password)) {
      errors.password = "Include at least one letter and one number";
    }

    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) return;

    await register({
      username: normalized,
      email: email.trim().toLowerCase(),
      password,
      ...(displayName.trim() ? { displayName: displayName.trim() } : {}),
    });
  }

  return (
    <main className="flex h-full w-full items-center justify-center overflow-y-auto bg-background p-6">
      <div className="w-full max-w-sm space-y-6">
        <div className="space-y-2 text-center">
          <h1 className="text-2xl font-bold">Create an account</h1>
          <p className="text-sm text-muted-foreground">Join Cordis in a few seconds</p>
        </div>

        <form onSubmit={(event) => void onSubmit(event)} className="space-y-4">
          <Field
            id="username"
            label="Username"
            value={username}
            onChange={setUsername}
            placeholder="yourname"
            autoComplete="username"
            error={fieldErrors.username}
          />
          <Field
            id="email"
            label="Email"
            type="email"
            value={email}
            onChange={setEmail}
            placeholder="you@example.com"
            autoComplete="email"
            error={fieldErrors.email}
          />
          <Field
            id="displayName"
            label="Display name"
            value={displayName}
            onChange={setDisplayName}
            placeholder="Optional"
            autoComplete="nickname"
            error={fieldErrors.displayName}
          />

          <div className="space-y-2">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              autoComplete="new-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
            {fieldErrors.password ? (
              <p className="text-xs text-destructive">{fieldErrors.password}</p>
            ) : (
              <p className="text-xs text-muted-foreground">
                At least 8 characters, with a letter and a number
              </p>
            )}
          </div>

          {error ? (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          <Button type="submit" className="w-full" disabled={isLoading}>
            {isLoading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            Create account
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          Already registered?{" "}
          <Link to="/login" className="font-medium text-primary hover:underline">
            Sign in
          </Link>
        </p>
      </div>
    </main>
  );
}

interface FieldProps {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: string;
  autoComplete?: string;
  error?: string;
}

function Field({
  id,
  label,
  value,
  onChange,
  placeholder,
  type = "text",
  autoComplete,
  error,
}: FieldProps): React.JSX.Element {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type={type}
        autoComplete={autoComplete}
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </div>
  );
}