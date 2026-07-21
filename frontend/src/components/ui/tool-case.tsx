'use client';
import type { Variants } from 'motion/react';
import type { HTMLAttributes } from 'react';
import { forwardRef, useCallback, useImperativeHandle, useRef } from 'react';
import { motion, useAnimation } from 'motion/react';
import { cn } from '@/lib/utils';
export interface ToolCaseIconHandle {
    startAnimation: () => void;
    stopAnimation: () => void;
}
interface ToolCaseIconProps extends HTMLAttributes<HTMLDivElement> {
    size?: number;
}
const DRAW_VARIANTS: Variants = {
    normal: {
        pathLength: 1,
        opacity: 1,
    },
    animate: {
        pathLength: [0, 1],
        opacity: [0, 1],
        transition: {
            duration: 0.6,
            ease: 'easeInOut',
        },
    },
};
const HANDLE_VARIANTS: Variants = {
    normal: {
        scaleX: 1,
        originX: '50%',
    },
    animate: {
        scaleX: [0.6, 1.1, 1],
        originX: '50%',
        transition: {
            duration: 0.45,
            ease: 'easeInOut',
        },
    },
};
const ToolCaseIcon = forwardRef<ToolCaseIconHandle, ToolCaseIconProps>(({ onMouseEnter, onMouseLeave, className, size = 28, ...props }, ref) => {
    const controls = useAnimation();
    const isControlledRef = useRef(false);
    useImperativeHandle(ref, () => {
        isControlledRef.current = true;
        return {
            startAnimation: () => controls.start('animate'),
            stopAnimation: () => controls.start('normal'),
        };
    });
    const handleMouseEnter = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
        if (!isControlledRef.current) {
            controls.start('animate');
        }
        else {
            onMouseEnter?.(e);
        }
    }, [controls, onMouseEnter]);
    const handleMouseLeave = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
        if (!isControlledRef.current) {
            controls.start('normal');
        }
        else {
            onMouseLeave?.(e);
        }
    }, [controls, onMouseLeave]);
    return (<div className={cn(className)} onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave} {...props}>
      <svg xmlns="http://www.w3.org/2000/svg" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <motion.path d="M10 15h4" variants={HANDLE_VARIANTS} animate={controls} initial="normal"/>
        <motion.path d="m14.817 10.995-.971-1.45 1.034-1.232a2 2 0 0 0-2.025-3.238l-1.82.364L9.91 3.885a2 2 0 0 0-3.625.748L6.141 6.55l-1.725.426a2 2 0 0 0-.19 3.756l.657.27" variants={DRAW_VARIANTS} animate={controls} initial="normal"/>
        <motion.path d="m18.822 10.995 2.26-5.38a1 1 0 0 0-.557-1.318L16.954 2.9a1 1 0 0 0-1.281.533l-.924 2.122" variants={DRAW_VARIANTS} animate={controls} initial="normal"/>
        <motion.path d="M4 12.006A1 1 0 0 1 4.994 11H19a1 1 0 0 1 1 1v7a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2z" variants={DRAW_VARIANTS} animate={controls} initial="normal"/>
      </svg>
    </div>);
});
ToolCaseIcon.displayName = 'ToolCaseIcon';
export { ToolCaseIcon };
