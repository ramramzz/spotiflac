'use client';
import type { Variants } from 'motion/react';
import type { HTMLAttributes } from 'react';
import { forwardRef, useCallback, useImperativeHandle, useRef } from 'react';
import { motion, useAnimation } from 'motion/react';
import { cn } from '@/lib/utils';
export interface FileTextIconHandle {
    startAnimation: () => void;
    stopAnimation: () => void;
}
interface FileTextIconProps extends HTMLAttributes<HTMLDivElement> {
    size?: number;
}
const PATH_VARIANTS: Variants = {
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
const FileTextIcon = forwardRef<FileTextIconHandle, FileTextIconProps>(({ onMouseEnter, onMouseLeave, className, size = 28, ...props }, ref) => {
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
                    <motion.path d="M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z" variants={PATH_VARIANTS} animate={controls} initial="normal"/>
                    <motion.path d="M14 2v5a1 1 0 0 0 1 1h5" variants={PATH_VARIANTS} animate={controls} initial="normal"/>
                    <motion.path d="M10 9H8" variants={PATH_VARIANTS} animate={controls} initial="normal"/>
                    <motion.path d="M16 13H8" variants={PATH_VARIANTS} animate={controls} initial="normal"/>
                    <motion.path d="M16 17H8" variants={PATH_VARIANTS} animate={controls} initial="normal"/>
                </svg>
            </div>);
});
FileTextIcon.displayName = 'FileTextIcon';
export { FileTextIcon };
