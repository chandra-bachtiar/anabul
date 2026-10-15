/**
 * Vibrate only when the device supports it and the user has not asked for
 * reduced motion.
 */
function canVibrate(): boolean {
	if (typeof navigator === 'undefined' || typeof navigator.vibrate !== 'function') {
		return false;
	}
	if (typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
		return false;
	}
	return true;
}

export function hapticLight(): void {
	if (!canVibrate()) return;
	navigator.vibrate(12);
}

export function hapticCelebrate(): void {
	if (!canVibrate()) return;
	navigator.vibrate([16, 60, 24]);
}

export function motionAllowed(): boolean {
	if (typeof window === 'undefined') return false;
	return !window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}
