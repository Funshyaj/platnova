import { NavLink } from 'react-router-dom'

const linkBase = 'block rounded-md px-3 py-2 text-sm font-medium'
const linkActive = 'bg-blue-600 text-white'
const linkInactive = 'text-gray-300 hover:bg-gray-800 hover:text-white'

export default function Sidebar() {
  return (
    <aside className="w-52 shrink-0 bg-gray-900 px-4 py-6 text-white">
      <div className="mb-8 text-xl font-bold">Platnova</div>
      <nav className="flex flex-col gap-1">
        <NavLink to="/home" className={({ isActive }) => `${linkBase} ${isActive ? linkActive : linkInactive}`}>
          Home
        </NavLink>
        <NavLink to="/transfer" className={({ isActive }) => `${linkBase} ${isActive ? linkActive : linkInactive}`}>
          Transfer
        </NavLink>
      </nav>
    </aside>
  )
}
