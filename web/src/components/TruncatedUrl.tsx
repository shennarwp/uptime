import { useState, useEffect } from 'react';

function useIsMobile() {
  const [isMobile, setIsMobile] = useState(() => window.innerWidth <= 768);

  useEffect(() => {
    const handler = () => setIsMobile(window.innerWidth <= 768);
    window.addEventListener('resize', handler);
    return () => window.removeEventListener('resize', handler);
  }, []);

  return isMobile;
}

export function TruncatedUrl({ url }: { url: string }) {
  const [isHovered, setIsHovered] = useState(false);
  const isMobile = useIsMobile();
  const maxLength = isMobile ? 15 : 30;

  const cleanUrl = url.replace(/^https?:\/\//, '');
  const displayUrl =
    isHovered || cleanUrl.length <= maxLength ? cleanUrl : cleanUrl.slice(0, maxLength) + '...';

  return (
    <a
      href={url}
      target="_blank"
      rel="noreferrer"
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      title={url}
      className={`truncated-url ${isHovered ? 'expanded' : ''}`}
    >
      {displayUrl} ↗
    </a>
  );
}
