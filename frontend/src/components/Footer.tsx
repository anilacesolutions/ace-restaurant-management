// Global builder credit — light, sits at the bottom of every surface.
export function Footer() {
  return (
    <footer className="mt-auto flex flex-col items-center gap-2 border-t border-zinc-200 px-4 py-6 text-center">
      <a
        href="https://www.ace-solutions.io"
        target="_blank"
        rel="noopener noreferrer"
        className="flex flex-col items-center gap-2"
      >
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src="/ace-desktop-logo-light.svg"
          alt="ACE Solutions"
          className="h-6 w-auto"
        />
        <span className="text-xs text-zinc-400">
          Made with <span className="text-red-500">♥</span> by ACE Solutions
        </span>
      </a>
    </footer>
  );
}
