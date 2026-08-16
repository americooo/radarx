interface RadarIconProps {
    size?: number;
    className?: string;
    animated?: boolean;
}

// Self-contained radar glyph: outer scope rings, a sweeping gradient wedge,
// and a pulsing center dot. No external assets required.
function RadarIcon({size = 24, className = '', animated = false}: RadarIconProps) {
    return (
        <svg
            width={size}
            height={size}
            viewBox="0 0 100 100"
            className={className}
            role="img"
            aria-label="RadarX"
        >
            <defs>
                <radialGradient id="radarSweepGradient" cx="50%" cy="50%" r="50%">
                    <stop offset="0%" stopColor="#60a5fa" stopOpacity="0.9" />
                    <stop offset="100%" stopColor="#60a5fa" stopOpacity="0" />
                </radialGradient>
            </defs>

            <circle cx="50" cy="50" r="46" fill="none" stroke="#1d4ed8" strokeOpacity="0.35" strokeWidth="1.5" />
            <circle cx="50" cy="50" r="32" fill="none" stroke="#1d4ed8" strokeOpacity="0.35" strokeWidth="1.5" />
            <circle cx="50" cy="50" r="18" fill="none" stroke="#1d4ed8" strokeOpacity="0.35" strokeWidth="1.5" />
            <line x1="4" y1="50" x2="96" y2="50" stroke="#1d4ed8" strokeOpacity="0.2" strokeWidth="1" />
            <line x1="50" y1="4" x2="50" y2="96" stroke="#1d4ed8" strokeOpacity="0.2" strokeWidth="1" />

            <path
                d="M 50 50 L 50 4 A 46 46 0 0 1 90.8 27 Z"
                fill="url(#radarSweepGradient)"
                className={animated ? 'origin-center animate-spin' : ''}
                style={animated ? {transformOrigin: '50px 50px', animationDuration: '3s'} : undefined}
            />

            <circle cx="50" cy="50" r="4" fill="#3b82f6" className={animated ? 'animate-pulse' : ''} />
            {animated && (
                <circle cx="50" cy="50" r="4" fill="none" stroke="#60a5fa" strokeWidth="2" className="animate-ping" />
            )}
        </svg>
    );
}

export default RadarIcon;
