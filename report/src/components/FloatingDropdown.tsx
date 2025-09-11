import React, { useEffect, useRef, useState, useCallback } from 'react';
import { createPortal } from 'react-dom';
import clsx from 'clsx';

interface FloatingDropdownProps {
  trigger: React.ReactNode;
  children: React.ReactNode;
  isOpen: boolean;
  onToggle: () => void;
  onClose: () => void;
  className?: string;
  dropdownClassName?: string;
  placement?: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left';
}

interface Position {
  top: number;
  left: number;
  maxHeight?: number;
}

const FloatingDropdown: React.FC<FloatingDropdownProps> = ({
  trigger,
  children,
  isOpen,
  onToggle,
  onClose,
  className = '',
  dropdownClassName = '',
  placement = 'bottom-right',
}) => {
  const triggerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState<Position>({ top: 0, left: 0 });

  // Calculate dropdown position
  const calculatePosition = useCallback(() => {
    if (!triggerRef.current) return;

    const triggerRect = triggerRef.current.getBoundingClientRect();
    const viewport = {
      width: window.innerWidth,
      height: window.innerHeight,
    };

    let top = 0;
    let left = 0;
    let maxHeight: number | undefined;

    // Calculate horizontal position
    switch (placement) {
      case 'bottom-right':
      case 'top-right':
        left = triggerRect.right - 200; // Assume dropdown width of 200px
        if (left < 10) left = 10; // Minimum margin from left edge
        break;
      case 'bottom-left':
      case 'top-left':
        left = triggerRect.left;
        if (left + 200 > viewport.width - 10) {
          left = viewport.width - 210; // Minimum margin from right edge
        }
        break;
    }

    // Calculate vertical position
    switch (placement) {
      case 'bottom-right':
      case 'bottom-left': {
        top = triggerRect.bottom + 8;
        // Check if dropdown would go below viewport
        const spaceBelow = viewport.height - top - 20; // 20px margin
        if (spaceBelow < 100) {
          // Not enough space below, position above
          top = triggerRect.top - 8;
          maxHeight = triggerRect.top - 20;
        } else {
          maxHeight = spaceBelow;
        }
        break;
      }
      case 'top-right':
      case 'top-left': {
        top = triggerRect.top - 8;
        maxHeight = triggerRect.top - 20;
        if (maxHeight < 100) {
          // Not enough space above, position below
          top = triggerRect.bottom + 8;
          maxHeight = viewport.height - top - 20;
        }
        break;
      }
    }

    setPosition({ top, left, maxHeight });
  }, [placement]);

  // Update position when dropdown opens or window resizes
  useEffect(() => {
    if (isOpen) {
      calculatePosition();
      
      const handleResize = () => calculatePosition();
      const handleScroll = () => calculatePosition();
      
      window.addEventListener('resize', handleResize);
      window.addEventListener('scroll', handleScroll, true);
      
      return () => {
        window.removeEventListener('resize', handleResize);
        window.removeEventListener('scroll', handleScroll, true);
      };
    }
  }, [isOpen, calculatePosition]);

  // Handle click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      
      if (
        triggerRef.current?.contains(target) ||
        dropdownRef.current?.contains(target)
      ) {
        return;
      }
      
      onClose();
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen, onClose]);

  // Handle escape key
  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      return () => document.removeEventListener('keydown', handleEscape);
    }
  }, [isOpen, onClose]);

  const dropdownContent = isOpen ? (
    <div
      ref={dropdownRef}
      className={clsx(
        'fixed z-50 bg-white border border-slate-200 rounded-md shadow-lg',
        'min-w-48 overflow-hidden',
        dropdownClassName
      )}
      style={{
        top: position.top,
        left: position.left,
        maxHeight: position.maxHeight ? `${position.maxHeight}px` : undefined,
      }}
    >
      <div className="py-1 overflow-y-auto max-h-full">
        {children}
      </div>
    </div>
  ) : null;

  return (
    <>
      <div ref={triggerRef} className={clsx('relative', className)}>
        <div onClick={onToggle} role="button" tabIndex={0} onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            onToggle();
          }
        }}>
          {trigger}
        </div>
      </div>
      {dropdownContent && createPortal(dropdownContent, document.body)}
    </>
  );
};

export default FloatingDropdown;
