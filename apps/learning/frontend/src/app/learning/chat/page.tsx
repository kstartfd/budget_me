import { Suspense } from "react"
import { TeacherChatScreen } from "@/features/chat"

export default function ChatPage() {
  return (
    <Suspense>
      <TeacherChatScreen />
    </Suspense>
  )
}
