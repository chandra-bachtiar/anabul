<script lang="ts">
	interface Props {
		mood?: 'happy' | 'worried' | 'asleep';
		size?: number;
		alt?: string;
		class?: string;
	}

	let { mood = 'happy', size = 64, alt, class: className = '' }: Props = $props();

	const sources: Record<string, string> = {
		happy: '/mascot-happy.svg',
		worried: '/mascot-worried.svg',
		asleep: '/mascot-asleep.svg'
	};

	const altTexts: Record<string, string> = {
		happy: 'Maskot ANABUL sedang senang',
		worried: 'Maskot ANABUL khawatir karena budget menipis',
		asleep: 'Maskot ANABUL tidur'
	};
</script>

<img
	src={sources[mood] ?? sources.happy}
	width={size}
	height={size}
	alt={alt ?? altTexts[mood] ?? ''}
	class="shrink-0 select-none {className}"
/>
