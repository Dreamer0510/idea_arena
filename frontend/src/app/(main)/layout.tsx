import { AppHeader } from "@/components/layout/app-header";

export default function MainLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <AppHeader />
      <main className="container mx-auto px-4 py-8 pb-20 md:pb-8">{children}</main>
    </>
  );
}
