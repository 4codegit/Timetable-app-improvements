<script>
        export let activeTab;
        export let onChange;

        const tabs = [
                { id: "refs", icon: "📚", label: "Справочники" },
                { id: "lessons", icon: "📝", label: "Учебный план" },
                { id: "constraints", icon: "⚠️", label: "Ограничения" },
                { id: "settings", icon: "⚙️", label: "Настройки" },
                { id: "schedule", icon: "🗓️", label: "Расписание" },
                { id: "issues", icon: "🐛", label: "Проблемы" }
        ];
</script>

<aside class="sidebar">
        <div class="brand">📅 <span>Timetable</span></div>
        <nav class="nav" aria-label="Разделы приложения">
                {#each tabs as item}
                        <button
                                class:active={activeTab === item.id}
                                aria-current={activeTab === item.id ? "page" : undefined}
                                on:click={() => onChange(item.id)}
                        >
                                <span class="ico">{item.icon}</span>{item.label}
                        </button>
                {/each}
        </nav>
</aside>
