import { cubicOut } from 'svelte/easing';

export const ease = cubicOut;

export const dur = 180;
export const durFast = 120;
export const durCelebrate = 420;

export const rise = { y: 4, duration: dur, easing: ease };
export const riseFast = { y: 4, duration: durFast, easing: ease };
export const fade = { duration: dur, easing: ease };
export const fadeFast = { duration: durFast, easing: ease };

export const springCelebrate = { stiffness: 0.18, damping: 0.55 };
