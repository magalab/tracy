/**
 * Copy text in both secure contexts and HTTP deployments.
 *
 * navigator.clipboard is unavailable in browsers when the app is served over
 * plain HTTP (a common setup for the Docker image). The textarea fallback is
 * deprecated, but is retained for HTTP deployments and must run during the
 * user's click gesture.
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Some browsers expose the API but reject the operation. Try the
      // compatibility path before reporting failure to the caller.
    }
  }

  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  textarea.style.pointerEvents = "none";
  document.body.appendChild(textarea);
  textarea.select();

  let copied = false;
  try {
    copied = document.execCommand("copy");
  } catch {
    copied = false;
  } finally {
    textarea.remove();
  }
  return copied;
}
