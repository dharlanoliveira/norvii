import { MarkdownTextPrimitive } from "@assistant-ui/react-markdown";
import remarkGfm from "remark-gfm";

export function AssistantMarkdown() {
  return (
    <MarkdownTextPrimitive
      className="message-markdown"
      defer
      remarkPlugins={[remarkGfm]}
      smooth={false}
    />
  );
}
